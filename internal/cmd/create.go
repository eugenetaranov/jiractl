package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	jiralib "github.com/andygrunwald/go-jira"
	"github.com/eugenetaranov/jiractl/internal/config"
	"github.com/eugenetaranov/jiractl/internal/jira"
	"github.com/eugenetaranov/jiractl/internal/textutil"
	"github.com/eugenetaranov/jiractl/internal/tui"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Jira issue",
	Long: `Create a new Jira issue, interactively or from flags.

Interactively, you're asked for the summary, description and epic, then shown
every field that will be sent. At "Create? [Y/e/d/n]", e edits a field, d
opens the description in $EDITOR, and n saves a draft and cancels.

With -y, or when stdin is not a terminal, nothing is asked: the issue is
created from flags and defaults, and only its key is printed to stdout.`,
	Example: `  jiractl create
  jiractl create -s "Fix login" -t Bug -y
  echo "Steps to reproduce..." | jiractl create -s "Fix login" -d - -F "Story Points=3"`,
	Args: cobra.NoArgs,
	RunE: runCreate,
}

// createOptions holds the create command's flags.
type createOptions struct {
	summary     string
	issueType   string
	epic        string
	description string
	fields      []string
	yes         bool
	noEpic      bool
}

var createOpts createOptions

func init() {
	f := createCmd.Flags()
	f.StringVarP(&createOpts.summary, "summary", "s", "", "Issue summary")
	f.StringVarP(&createOpts.issueType, "type", "t", "", "Issue type (default: issue_defaults.issue_type)")
	f.StringVarP(&createOpts.epic, "epic", "e", "", "Epic key (default: issue_defaults.epic_link)")
	f.StringVarP(&createOpts.description, "description", "d", "", `Description; "-" reads it from stdin`)
	f.StringArrayVarP(&createOpts.fields, "field", "F", nil, "Set a field by name or ID: name=value (repeatable)")
	f.BoolVarP(&createOpts.yes, "yes", "y", false, "Create without prompts or confirmation")
	f.BoolVar(&createOpts.noEpic, "no-epic", false, "Create the issue without an epic, ignoring the default")
	RootCmd.AddCommand(createCmd)
}

type epicCheckResult struct {
	state jira.EpicState
	issue *jiralib.Issue
	err   error
}

// createPrefetch holds everything create may need from Jira, fetched in the
// background as soon as the command starts.
type createPrefetch struct {
	issueTypes  *pending[[]jiralib.IssueType]
	epics       *pending[[]jiralib.Issue]
	fields      *pending[*fieldNames]
	defaultEpic *pending[epicCheckResult]
	assignee    *pending[string]
}

func startPrefetch(cfg *config.Config, client *jira.Client, epicKey, assignee string) *createPrefetch {
	pf := &createPrefetch{
		issueTypes: fetch(func() ([]jiralib.IssueType, error) { return client.GetIssueTypes(cfg.Project) }),
		epics:      fetch(func() ([]jiralib.Issue, error) { return client.GetEpics(cfg.Project) }),
		fields: fetch(func() (*fieldNames, error) {
			fields, err := client.GetFields()
			if err != nil {
				return nil, err
			}
			return newFieldNames(fields), nil
		}),
		defaultEpic: ready(epicCheckResult{}),
		assignee:    ready(""),
	}
	if epicKey != "" {
		pf.defaultEpic = lookupEpic(client, epicKey)
	}
	if assignee != "" && !cfg.IsServer() {
		pf.assignee = fetch(func() (string, error) { return client.ResolveAccountID(assignee) })
	}
	return pf
}

// applyDefaults copies issue_defaults into a fresh draft.
func applyDefaults(draft *issueDraft, cfg *config.Config) {
	if draft.DefaultsApplied {
		return
	}
	d := cfg.IssueDefaults
	if draft.IssueType == "" {
		draft.IssueType = d.IssueType
	}
	if draft.EpicLink == "" {
		draft.EpicLink = d.EpicLink
	}
	draft.Assignee = d.Assignee
	draft.Component = d.Component
	draft.Labels = append([]string(nil), d.Labels...)
	fields := map[string]string{}
	for k, v := range d.CustomFields {
		fields[k] = v
	}
	for k, v := range draft.Fields {
		fields[k] = v
	}
	draft.Fields = fields
	draft.DefaultsApplied = true
}

func runCreate(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	client, err := jira.NewClient(cfg)
	if err != nil {
		return err
	}

	o := createOpts
	interactive := !o.yes && isInteractive()
	if o.description == "-" {
		// stdin carries the description, so it can't also answer prompts.
		interactive = false
	}

	defaultEpic := cfg.IssueDefaults.EpicLink
	if o.noEpic || o.epic != "" {
		defaultEpic = ""
	}
	pf := startPrefetch(cfg, client, defaultEpic, cfg.IssueDefaults.Assignee)

	// Merge order: defaults -> saved draft -> flags -> prompts -> edits.
	draft := &issueDraft{Project: cfg.Project}
	resumed := false
	hasFlags := o.summary != "" || o.issueType != "" || o.epic != "" || o.description != "" || len(o.fields) > 0
	if interactive && !hasFlags {
		if saved := loadDraft(); saved != nil && saved.Project == cfg.Project {
			label := fmt.Sprintf("Resume draft %q from %s?", textutil.Truncate(saved.Summary, 40), saved.SavedAt.Format("Jan 2 15:04"))
			resume, err := promptConfirm(label, true)
			if err != nil {
				return err
			}
			if resume {
				draft, resumed = saved, true
			} else {
				deleteDraft()
			}
		}
	}
	applyDefaults(draft, cfg)

	if err := applyCreateFlags(draft, o, pf); err != nil {
		return err
	}

	if interactive {
		if err := promptMissing(cfg, client, draft, pf, resumed, o); err != nil {
			return err
		}
	}
	if draft.Summary == "" {
		return errors.New("summary required: pass -s")
	}
	if draft.IssueType == "" {
		return errors.New("issue type required: pass -t or set issue_defaults.issue_type")
	}
	if err := normalizeIssueType(draft, pf, interactive); err != nil {
		return err
	}

	// Make sure the epic can be used before anything is sent.
	var epic *jiralib.Issue
	if draft.EpicLink != "" {
		var res epicCheckResult
		if draft.EpicLink == defaultEpic {
			res, _ = pf.defaultEpic.wait("epic")
		} else {
			res.state, res.issue, res.err = client.CheckEpic(draft.EpicLink)
		}
		if problem := epicProblem(res.state); problem != "" {
			epic, err = recoverEpic(cfg, client, draft, problem, draft.EpicLink == cfg.IssueDefaults.EpicLink, interactive)
			if err != nil {
				keepDraft(draft)
				return err
			}
		} else {
			if res.err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not check epic %s: %v\n", draft.EpicLink, res.err)
			}
			epic = res.issue
		}
	}

	names, _ := pf.fields.wait("field names")

	if interactive {
		confirmed, err := reviewLoop(cfg, client, draft, pf, names, &epic)
		if err != nil {
			keepDraft(draft)
			return err
		}
		if !confirmed {
			keepDraft(draft)
			return ErrCancelled
		}
	}

	var issue *jiralib.Issue
	for attempt := 0; ; attempt++ {
		payload, err := buildPayload(cfg, client, draft, pf)
		if err == nil {
			issue, err = client.CreateIssue(payload)
		}
		if err == nil {
			break
		}
		// Jira can still refuse the epic as parent; let the user pick again.
		if attempt < 2 && draft.EpicLink != "" && jira.IsEpicRejection(err) && interactive {
			problem := fmt.Sprintf("was rejected by Jira (%v)", err)
			if _, rerr := recoverEpic(cfg, client, draft, problem, draft.EpicLink == cfg.IssueDefaults.EpicLink, interactive); rerr != nil {
				keepDraft(draft)
				return rerr
			}
			continue
		}
		keepDraft(draft)
		return fmt.Errorf("failed to create issue: %w", err)
	}
	deleteDraft()

	url := fmt.Sprintf("%s/browse/%s", cfg.Server, issue.Key)
	menuStatus = fmt.Sprintf("Created %s  %s", issue.Key, url)
	if interactive {
		fmt.Printf("\nCreated issue: %s\n%s\n", issue.Key, url)
	} else {
		fmt.Println(issue.Key)
		fmt.Fprintln(os.Stderr, url)
	}
	return nil
}

func applyCreateFlags(draft *issueDraft, o createOptions, pf *createPrefetch) error {
	if o.issueType != "" {
		draft.IssueType = o.issueType
	}
	if o.summary != "" {
		draft.Summary = o.summary
	}
	switch o.description {
	case "":
	case "-":
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read description from stdin: %w", err)
		}
		draft.Description = strings.TrimRight(string(data), "\r\n")
	default:
		draft.Description = o.description
	}
	if o.epic != "" {
		draft.EpicLink = strings.ToUpper(strings.TrimSpace(o.epic))
	}
	if o.noEpic {
		draft.EpicLink = ""
	}
	if len(o.fields) > 0 {
		names, err := pf.fields.wait("field names")
		if err != nil {
			names = nil
		}
		for _, f := range o.fields {
			key, value, err := parseFieldFlag(f)
			if err != nil {
				return err
			}
			id, err := names.resolve(key)
			if err != nil {
				return err
			}
			draft.Fields[id] = value
		}
	}
	return nil
}

// promptMissing asks for whatever the flags, defaults and draft left open.
func promptMissing(cfg *config.Config, client *jira.Client, draft *issueDraft, pf *createPrefetch, resumed bool, o createOptions) error {
	if draft.IssueType == "" {
		t, err := pickIssueType(pf, "Select issue type")
		if err != nil {
			return err
		}
		draft.IssueType = t
	}
	if resumed || o.summary != "" {
		return nil
	}

	summary, err := promptText("Summary", true)
	if err != nil {
		return err
	}
	draft.Summary = summary

	if o.description == "" {
		description, err := promptMultilineText("Description (optional)")
		if err != nil {
			return err
		}
		draft.Description = description
	}

	if draft.EpicLink == "" && !o.noEpic && o.epic == "" {
		epic, err := pickEpic(client, cfg.Project, pf.epics)
		if err != nil {
			return err
		}
		if epic != nil {
			draft.EpicLink = epic.Key
		}
	}
	return nil
}

func pickIssueType(pf *createPrefetch, header string) (string, error) {
	issueTypes, err := pf.issueTypes.wait("issue types")
	if err != nil {
		return "", fmt.Errorf("failed to get issue types: %w", err)
	}
	typeNames := make([]string, len(issueTypes))
	for i, it := range issueTypes {
		typeNames[i] = it.Name
	}
	idx, err := fzfSelect(typeNames, header)
	if err != nil {
		return "", err
	}
	return typeNames[idx], nil
}

// normalizeIssueType matches the type case-insensitively against the
// project's types, so "bug" becomes "Bug".
func normalizeIssueType(draft *issueDraft, pf *createPrefetch, interactive bool) error {
	issueTypes, err := pf.issueTypes.wait("issue types")
	if err != nil || len(issueTypes) == 0 {
		return nil // let Jira decide
	}
	var names []string
	for _, it := range issueTypes {
		if strings.EqualFold(it.Name, draft.IssueType) {
			draft.IssueType = it.Name
			return nil
		}
		names = append(names, it.Name)
	}
	if interactive {
		fmt.Fprintf(os.Stderr, "Issue type %q doesn't exist in this project.\n", draft.IssueType)
		t, err := pickIssueType(pf, "Select issue type")
		if err != nil {
			return err
		}
		draft.IssueType = t
		return nil
	}
	return fmt.Errorf("issue type %q doesn't exist in this project; available: %s", draft.IssueType, strings.Join(names, ", "))
}

// buildPayload turns the draft into exactly what will be sent.
func buildPayload(cfg *config.Config, client *jira.Client, draft *issueDraft, pf *createPrefetch) (*jira.NewIssue, error) {
	n := &jira.NewIssue{
		Project:     cfg.Project,
		Type:        draft.IssueType,
		Summary:     draft.Summary,
		Description: draft.Description,
		EpicLink:    draft.EpicLink,
		Component:   draft.Component,
		Labels:      draft.Labels,
		Fields:      draft.Fields,
	}
	if draft.EpicLink != "" {
		n.EpicField = client.EpicField(cfg.Project, draft.IssueType)
		if n.EpicField == "" {
			fmt.Fprintf(os.Stderr, "Warning: %s issues in %s have no parent or Epic Link field; creating without epic %s (set issue_defaults.epic_field to override)\n",
				draft.IssueType, cfg.Project, draft.EpicLink)
			n.EpicLink = ""
		}
	}
	if draft.Assignee != "" && cfg.IsServer() {
		// Server/Data Center identifies users by username.
		n.AssigneeName = draft.Assignee
	} else if draft.Assignee != "" {
		var id string
		var err error
		if draft.Assignee == cfg.IssueDefaults.Assignee {
			id, err = pf.assignee.wait("assignee")
		} else {
			id, err = client.ResolveAccountID(draft.Assignee)
		}
		if err != nil {
			return nil, err
		}
		n.AccountID = id
	}
	return n, nil
}

type reviewRow struct {
	label, value string
}

func reviewRows(draft *issueDraft, names *fieldNames, epic *jiralib.Issue, defaultEpic string) []reviewRow {
	rows := []reviewRow{
		{"Type", draft.IssueType},
		{"Summary", draft.Summary},
	}
	if draft.Description != "" {
		rows = append(rows, reviewRow{"Description", draft.Description})
	}
	epicValue := draft.EpicLink
	switch {
	case draft.EpicLink == "":
		epicValue = "(none)"
	case epic != nil && epic.Key == draft.EpicLink && epicSummary(epic) != "":
		epicValue += " - " + epicSummary(epic)
	}
	if draft.EpicLink != "" && draft.EpicLink == defaultEpic {
		epicValue += " (default)"
	}
	rows = append(rows, reviewRow{"Epic", epicValue})
	if draft.Assignee != "" {
		rows = append(rows, reviewRow{"Assignee", draft.Assignee})
	}
	if draft.Component != "" {
		rows = append(rows, reviewRow{"Component", draft.Component})
	}
	if len(draft.Labels) > 0 {
		rows = append(rows, reviewRow{"Labels", strings.Join(draft.Labels, ", ")})
	}
	ids := make([]string, 0, len(draft.Fields))
	for id := range draft.Fields {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return names.label(ids[i]) < names.label(ids[j]) })
	for _, id := range ids {
		rows = append(rows, reviewRow{names.label(id), draft.Fields[id]})
	}
	return rows
}

// reviewLoop shows the payload and handles Create? [Y/e/d/n]. It returns
// false when the user answers n.
func reviewLoop(cfg *config.Config, client *jira.Client, draft *issueDraft, pf *createPrefetch, names *fieldNames, epic **jiralib.Issue) (bool, error) {
	for {
		rows := reviewRows(draft, names, *epic, cfg.IssueDefaults.EpicLink)
		review := make([]tui.ReviewRow, len(rows))
		for i, r := range rows {
			review[i] = tui.ReviewRow{Label: r.label, Value: r.value}
		}
		choice, err := tui.Review("Creating issue in "+cfg.Project+":", review, "Create?", "yedn")
		if err != nil {
			return false, err
		}
		switch choice {
		case 'y':
			return true, nil
		case 'n':
			return false, nil
		case 'd':
			desc, err := openEditor(draft.Description)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			draft.Description = desc
		case 'e':
			if err := editField(cfg, client, draft, pf, names, epic); err != nil {
				return false, err
			}
		}
	}
}

const addFieldRow = "Add field…"

// editField lets the user change one field of the draft. Esc returns to the
// review without changes.
func editField(cfg *config.Config, client *jira.Client, draft *issueDraft, pf *createPrefetch, names *fieldNames, epic **jiralib.Issue) error {
	rows := []string{"Type", "Summary", "Description", "Epic", "Assignee", "Component", "Labels"}
	ids := make([]string, 0, len(draft.Fields))
	for id := range draft.Fields {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return names.label(ids[i]) < names.label(ids[j]) })
	for _, id := range ids {
		rows = append(rows, names.label(id))
	}
	rows = append(rows, addFieldRow)

	idx, err := fzfSelect(rows, "Edit which field? (Esc to go back)")
	if tui.IsEsc(err) {
		return nil
	}
	if err != nil {
		return err
	}

	const clearHint = ` ("-" clears)`
	text := func(label, cur string) (string, error) {
		v, err := promptTextWithDefault(label+clearHint, cur, false)
		if v == "-" {
			v = ""
		}
		return v, err
	}

	switch rows[idx] {
	case "Type":
		t, err := pickIssueType(pf, "Select issue type")
		if tui.IsEsc(err) {
			return nil
		}
		if err != nil {
			return err
		}
		draft.IssueType = t
	case "Summary":
		draft.Summary, err = promptTextWithDefault("Summary", draft.Summary, true)
	case "Description":
		var desc string
		desc, err = openEditor(draft.Description)
		if err == nil {
			draft.Description = desc
		}
	case "Epic":
		var e *jiralib.Issue
		e, err = pickEpic(client, cfg.Project, pf.epics)
		if tui.IsEsc(err) {
			return nil
		}
		if err == nil {
			*epic = e
			draft.EpicLink = ""
			if e != nil {
				draft.EpicLink = e.Key
			}
		}
	case "Assignee":
		draft.Assignee, err = text("Assignee (email, name or account ID)", draft.Assignee)
	case "Component":
		draft.Component, err = text("Component", draft.Component)
	case "Labels":
		var v string
		v, err = text("Labels (comma-separated)", strings.Join(draft.Labels, ", "))
		draft.Labels = nil
		for _, l := range strings.Split(v, ",") {
			if l = strings.TrimSpace(l); l != "" {
				draft.Labels = append(draft.Labels, l)
			}
		}
	case addFieldRow:
		var key, id string
		key, err = promptText("Field name or ID", true)
		if err != nil {
			return err
		}
		if id, err = names.resolve(key); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return nil
		}
		var v string
		v, err = promptText(names.label(id), true)
		if err == nil {
			draft.Fields[id] = v
		}
	default:
		id := ids[idx-7]
		var v string
		v, err = text(names.label(id), draft.Fields[id])
		if err == nil {
			if v == "" {
				delete(draft.Fields, id)
			} else {
				draft.Fields[id] = v
			}
		}
	}
	return err
}
