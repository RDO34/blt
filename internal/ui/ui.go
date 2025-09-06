package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rdo34/blt/internal/app"
	"github.com/rdo34/blt/internal/model"
	"github.com/rdo34/blt/internal/store"
	"github.com/rivo/tview"
)

const DefaultControls = "[a] Add  [e] Edit  [x] Delete  [t] Type  [#] Tags  [c] Complete  [m] Migrate  [s] Schedule  [j/k] Move  [?] Help"

type UI struct {
	app         *tview.Application
	grid        *tview.Grid
	list        *tview.List
	state       *app.App
	emptyState  bool
	pages       *tview.Pages
	title       tview.Primitive
	titleLeft   *tview.TextView
	titleRight  *tview.TextView
	controls    *tview.TextView
	sidebar     *tview.TextView
	selID       string
	selDate     time.Time
	centerWidth int

	// Inline input area (bottom, above controls)
	inputActive    bool
	inputPrimitive tview.Primitive
	// lightweight confirmation mode (no input box)
	confirmCallback func(confirm bool)
	promptMessage   string
}

// New constructs the TUI with a title bar and controls footer.
func New() *UI {
	appView := tview.NewApplication()

	// State and storage
	st, err := store.NewDefaultFSStore()
	if err != nil {
		// Fallback to a local workspace directory when OS dirs are unavailable.
		st, _ = store.NewFSStore(".blt-data")
	}
	state := app.New(st)
	// Load preferences if available
	centerWidth := 80
	if prefs, err := store.LoadPreferences(); err == nil {
		if prefs.LastDate != "" {
			if d, err := time.Parse("2006-01-02", prefs.LastDate); err == nil {
				state.JumpToDate(d)
			}
		}
		if prefs.Period != "" {
			state.SetPeriod(prefs.Period)
		}
		if prefs.TextFilter != "" {
			state.SetTextFilter(prefs.TextFilter)
		}
		if len(prefs.Types) > 0 {
			state.SetTypeFilter(prefs.Types)
		}
		if len(prefs.Tags) > 0 {
			state.SetTagFilter(prefs.Tags)
		}
		if prefs.CenterWidth > 0 {
			centerWidth = prefs.CenterWidth
		}
		_ = state.Refresh()
	} else {
		_ = state.LoadDay(time.Now())
	}

	// Components
	// Sidebar removed for a more minimal layout
	var sideBar *tview.TextView = nil

	// Title: container with left/right aligned text (bottom rule instead of full border)
	titleLeft := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	titleRight := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignRight)
	titleGrid := tview.NewGrid().SetRows(1).SetColumns(0, 0)
	titleGrid.AddItem(titleLeft, 0, 0, 1, 1, 0, 0, false)
	titleGrid.AddItem(titleRight, 0, 1, 1, 1, 0, 0, false)
	// Remove full border; we'll draw a bottom rule below the title
	titleGrid.SetBorder(false)
	// Horizontal rule under header
	headerRule := tview.NewTextView().SetDynamicColors(true)
	headerRule.SetText("[green]" + strings.Repeat("─", 200))
	controls := tview.NewTextView().SetTextAlign(tview.AlignCenter)

	list := tview.NewList().ShowSecondaryText(false)

	// Center the list in a fixed-width middle column for a compact look
	grid := tview.NewGrid().
		SetRows(1, 1, 0, 0, 3).        // header, rule, list, (hidden input), controls
		SetColumns(0, centerWidth, 0). // left flex, center fixed width, right flex
		AddItem(titleGrid, 0, 0, 1, 3, 0, 0, false).
		AddItem(headerRule, 1, 0, 1, 3, 0, 0, false).
		AddItem(controls, 4, 0, 1, 3, 0, 0, false)

	grid.AddItem(list, 2, 1, 1, 1, 0, 0, true)

	u := &UI{app: appView, grid: grid, list: list, state: state, title: titleGrid, titleLeft: titleLeft, titleRight: titleRight, controls: controls, sidebar: sideBar}
	u.centerWidth = centerWidth
	u.refreshList()
	// Update controls when selection changes to reflect context-aware keybinds and track selection key
	list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		vis := u.state.Visible()
		if index >= 0 && index < len(vis) {
			u.selID = vis[index].Item.ID
			u.selDate = vis[index].Date
		}
		u.updateStatus()
	})

	// Global keybindings: quit, navigate, filters, scope, and CRUD
	grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// When input is active, either let focused input handle keys
		// or handle lightweight confirmation directly here.
		if u.inputActive {
			if u.confirmCallback != nil {
				switch event.Key() {
				case tcell.KeyEnter:
					cb := u.confirmCallback
					// Clear state before invoking
					u.confirmCallback = nil
					u.inputActive = false
					u.promptMessage = ""
					u.updateStatus()
					cb(true)
					return nil
				case tcell.KeyEscape:
					cb := u.confirmCallback
					u.confirmCallback = nil
					u.inputActive = false
					u.promptMessage = ""
					u.updateStatus()
					cb(false)
					return nil
				default:
					// Swallow all other keys while confirming
					return nil
				}
			}
			// For real input fields, allow the focused primitive to process
			return event
		}
		switch event.Key() {
		case tcell.KeyEscape:
			appView.Stop()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j':
				if !u.emptyState {
					moveDown(list)
				}
				return nil
			case 'k':
				if !u.emptyState {
					moveUp(list)
				}
				return nil
			case 'g':
				if !u.emptyState {
					moveHome(list)
				}
				return nil
			case 'G':
				if !u.emptyState {
					moveEnd(list)
				}
				return nil
			case '1':
				_ = u.state.SetPeriod(model.PeriodDay)
				u.refreshList()
				return nil
			case '2':
				_ = u.state.SetPeriod(model.PeriodWeek)
				u.refreshList()
				return nil
			case '3':
				_ = u.state.SetPeriod(model.PeriodMonth)
				u.refreshList()
				return nil
			case '[':
				_ = u.state.PrevPeriod()
				u.refreshList()
				return nil
			case ']':
				_ = u.state.NextPeriod()
				u.refreshList()
				return nil
			case 'd':
				u.showDateJump()
				return nil
			case 'T':
				_ = u.state.JumpToDate(time.Now())
				u.refreshList()
				return nil
			case '/':
				u.showTextFilter()
				return nil
			case ':':
				u.showTypeFilter()
				return nil
			case 'F':
				u.showTagFilter()
				return nil
			case '?':
				u.showHelp()
				return nil
			case 'a':
				if u.state.Period != model.PeriodDay {
					return nil
				}
				u.showAddDialog()
				return nil
			case 'e':
				if !u.emptyState {
					if u.state.Period != model.PeriodDay {
						return nil
					}
					u.showEditDialog()
				}
				return nil
			case 'x':
				if !u.emptyState {
					if u.state.Period != model.PeriodDay {
						return nil
					}
					u.showDeleteConfirm()
				}
				return nil
			case 'c':
				if !u.emptyState {
					if u.state.Period != model.PeriodDay {
						return nil
					}
					u.completeSelected()
				}
				return nil
			case 'm':
				if !u.emptyState {
					if u.state.Period != model.PeriodDay {
						return nil
					}
					u.migrateSelected()
				}
				return nil
			case 's':
				if !u.emptyState {
					if u.state.Period != model.PeriodDay {
						return nil
					}
					// If currently Scheduled, pressing 's' undoes scheduling without dialog
					idx := u.list.GetCurrentItem()
					vis := u.state.Visible()
					if idx >= 0 && idx < len(vis) && vis[idx].Item.Type == model.Scheduled {
						_ = u.state.ScheduleIndex(idx, time.Now())
						u.refreshList()
					} else {
						u.showScheduleDialog()
					}
				}
				return nil
			case 't':
				if !u.emptyState {
					if u.state.Period != model.PeriodDay {
						return nil
					}
					u.showTypePicker()
				}
				return nil
			case '#':
				if !u.emptyState {
					if u.state.Period != model.PeriodDay {
						return nil
					}
					u.showTagsDialog()
				}
				return nil
			}
		case tcell.KeyPgDn:
			if !u.emptyState {
				pageDown(list)
			}
			return nil
		case tcell.KeyPgUp:
			if !u.emptyState {
				pageUp(list)
			}
			return nil
		}
		return event
	})

	// Pages overlay for dialogs
	pages := tview.NewPages()
	pages.AddPage("main", grid, true, true)
	u.pages = pages

	return u
}

// Run starts the application event loop.
func (u *UI) Run() error {
	return u.app.SetRoot(u.pages, true).SetFocus(u.list).Run()
}

// refreshList rebuilds the list items from state and sets empty-state if needed.
func (u *UI) refreshList() {
	// Preserve current index and selection key
	prevIdx := u.list.GetCurrentItem()
	u.list.Clear()
	vis := u.state.Visible()
	u.updateSidebar()
	if len(vis) == 0 {
		u.list.AddItem("⟂ No items for today — press 'a' to add", "", 0, nil)
		u.emptyState = true
		u.list.SetCurrentItem(0)
		u.updateStatus()
		return
	}
	for _, e := range vis {
		u.list.AddItem(u.formatEntry(e), "", 0, nil)
	}
	u.emptyState = false
	// Determine best target index: by ID match, else previous index, else clamp
	target := 0
	if u.selID != "" {
		for i, e := range vis {
			if e.Item.ID == u.selID && e.Date.Equal(u.selDate) {
				target = i
				break
			}
		}
	}
	if target == 0 && prevIdx >= 0 {
		if prevIdx < len(vis) {
			target = prevIdx
		} else {
			target = len(vis) - 1
		}
	}
	u.list.SetCurrentItem(target)
	u.updateStatus()
}

func (u *UI) updateStatus() {
	// Title shows product name on left and scope+range+completion on right
	scope := strings.Title(string(u.state.Period))
	rng := u.stateVisibleRangeLabel()
	pct := u.percentComplete()
	left := "[red::b]BLT[-]"
	right := scope + " " + rng
	if pct >= 0 {
		right += "  (" + strconv.Itoa(pct) + "% done)"
	}
	if u.titleLeft != nil {
		u.titleLeft.SetText(left)
	}
	if u.titleRight != nil {
		u.titleRight.SetText(right)
	}

	filters := u.filtersSummary()
	note := ""
	if u.state.Period != model.PeriodDay {
		note = "    (Edits disabled in Week/Month — switch to Day)"
	}
	// When input is active, show only Enter/Esc hints
	if u.inputActive {
		// If in confirmation mode, include the prompt if any
		if u.confirmCallback != nil {
			msg := u.promptMessage
			if strings.TrimSpace(msg) != "" {
				u.controls.SetText(msg + "  [enter] Confirm  [esc] Cancel")
			} else {
				u.controls.SetText("[enter] Confirm  [esc] Cancel")
			}
		} else {
			u.controls.SetText("[enter] Confirm  [esc] Cancel")
		}
	} else {
		base := u.contextControls()
		// Hide Today hint when already at today (regardless of scope)
		today := time.Now()
		isToday := u.state.CurrentDate.Year() == today.Year() && u.state.CurrentDate.YearDay() == today.YearDay()
		if isToday {
			base = strings.Replace(base, "  [T] Today", "", 1)
		}
		u.controls.SetText(base + filters + note)
	}
}

func (u *UI) updateSidebar() {
	// Sidebar removed; keep method no-op for minimal changes
	if u.sidebar == nil {
		return
	}
}

func itoa(i int) string { return strconv.Itoa(i) }

func (u *UI) stateVisibleRangeLabel() string {
	switch u.state.Period {
	case model.PeriodWeek:
		r := u.stateVisibleRange()
		return r.Start.Format("2006-01-02") + " → " + r.End.Format("2006-01-02")
	case model.PeriodMonth:
		r := u.stateVisibleRange()
		return r.Start.Format("2006-01-02") + " → " + r.End.Format("2006-01-02")
	default:
		return u.state.CurrentDate.Format("2006-01-02")
	}
}

func (u *UI) stateVisibleRange() model.DateRange {
	// Recompute similarly to app.dateRange; duplicate to avoid export
	d := u.state.CurrentDate
	switch u.state.Period {
	case model.PeriodWeek:
		wd := int(d.Weekday())
		if wd == 0 {
			wd = 7
		}
		start := d.AddDate(0, 0, -(wd - 1))
		end := start.AddDate(0, 0, 6)
		return model.DateRange{Start: start, End: end}
	case model.PeriodMonth:
		start := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
		end := start.AddDate(0, 1, -1)
		return model.DateRange{Start: start, End: end}
	default:
		return model.DateRange{Start: d, End: d}
	}
}

func (u *UI) filtersSummary() string {
	var parts []string
	if q := u.state.TextFilter; q != "" {
		parts = append(parts, "/"+q)
	}
	if len(u.state.TypeFilter) > 0 {
		var ts []string
		for t := range u.state.TypeFilter {
			ts = append(ts, string(t))
		}
		parts = append(parts, ":"+strings.Join(ts, ","))
	}
	if len(u.state.TagFilter) > 0 {
		var ts []string
		for t := range u.state.TagFilter {
			ts = append(ts, "#"+t)
		}
		parts = append(parts, strings.Join(ts, " "))
	}
	if len(parts) == 0 {
		return ""
	}
	return "    Filters: " + strings.Join(parts, "  ")
}

// Dialogs and actions
func (u *UI) showAddDialog() {
	field := tview.NewInputField().SetLabel("Add: ").SetFieldWidth(60)
	field.SetBorder(false)
	styleInputField(field)
	field.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			u.hideInput()
			return nil
		case tcell.KeyEnter:
			text := field.GetText()
			if strings.TrimSpace(text) == "" {
				return nil
			}
			_, _ = u.state.Add(text)
			u.hideInput()
			u.refreshList()
			return nil
		}
		return event
	})
	u.showInput(field)
}

func (u *UI) showEditDialog() {
	idx := u.list.GetCurrentItem()
	vis := u.state.Visible()
	if idx < 0 || idx >= len(vis) {
		return
	}
	curr := vis[idx].Item
	field := tview.NewInputField().SetLabel("Edit: ").SetText(curr.Text).SetFieldWidth(60)
	field.SetBorder(false)
	styleInputField(field)
	field.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			u.hideInput()
			return nil
		case tcell.KeyEnter:
			if strings.TrimSpace(field.GetText()) == "" {
				return nil
			}
			_ = u.state.UpdateText(idx, field.GetText())
			u.hideInput()
			u.refreshList()
			return nil
		}
		return event
	})
	u.showInput(field)
}

func (u *UI) showDeleteConfirm() {
	// Lightweight confirm: no input box, only footer prompt and Enter/Esc
	u.inputActive = true
	u.promptMessage = "Delete selected item?"
	u.confirmCallback = func(confirm bool) {
		if confirm {
			idx := u.list.GetCurrentItem()
			_ = u.state.DeleteIndex(idx)
			u.refreshList()
		}
		// on cancel, nothing to do
	}
	u.updateStatus()
}

func (u *UI) completeSelected() {
	idx := u.list.GetCurrentItem()
	_ = u.state.CompleteIndex(idx)
	u.refreshList()
}

func (u *UI) migrateSelected() {
	idx := u.list.GetCurrentItem()
	_ = u.state.MigrateIndex(idx)
	u.refreshList()
}

func (u *UI) showScheduleDialog() {
	idx := u.list.GetCurrentItem()
	vis := u.state.Visible()
	if idx < 0 || idx >= len(vis) {
		return
	}
	field := tview.NewInputField().SetLabel("Schedule (YYYY-MM-DD): ").SetFieldWidth(20)
	field.SetBorder(false)
	styleInputField(field)
	field.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			u.hideInput()
			return nil
		case tcell.KeyEnter:
			txt := field.GetText()
			if d, err := time.Parse("2006-01-02", txt); err == nil {
				_ = u.state.ScheduleIndex(idx, d)
				u.hideInput()
				u.refreshList()
				return nil
			}
			return nil
		}
		return event
	})
	u.showInput(field)
}

func (u *UI) showDateJump() {
	field := tview.NewInputField().SetLabel("Date (YYYY-MM-DD): ").SetFieldWidth(20).SetText(u.state.CurrentDate.Format("2006-01-02"))
	field.SetBorder(false)
	styleInputField(field)
	field.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			u.hideInput()
			return nil
		case tcell.KeyEnter:
			txt := field.GetText()
			if d, err := time.Parse("2006-01-02", txt); err == nil {
				_ = u.state.JumpToDate(d)
				u.hideInput()
				u.refreshList()
				return nil
			}
			return nil
		}
		return event
	})
	u.showInput(field)
}

func (u *UI) showTextFilter() {
	field := tview.NewInputField().SetLabel("Text filter: ").SetFieldWidth(40).SetText(u.state.TextFilter)
	field.SetBorder(false)
	styleInputField(field)
	field.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			u.hideInput()
			return nil
		case tcell.KeyEnter:
			u.state.SetTextFilter(field.GetText())
			u.hideInput()
			u.refreshList()
			return nil
		}
		return event
	})
	u.showInput(field)
}

func (u *UI) showTypeFilter() {
	var preset []string
	for t, ok := range u.state.TypeFilter {
		if ok {
			preset = append(preset, string(t))
		}
	}
	field := tview.NewInputField().SetLabel("Types (comma): ").SetText(strings.Join(preset, ", ")).SetFieldWidth(60)
	field.SetBorder(false)
	styleInputField(field)
	field.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			u.hideInput()
			return nil
		case tcell.KeyEnter:
			raw := field.GetText()
			var types []model.BulletType
			for _, part := range strings.Split(raw, ",") {
				s := strings.TrimSpace(strings.ToLower(part))
				if s == "" {
					continue
				}
				switch s {
				case "task":
					types = append(types, model.Task)
				case "done":
					types = append(types, model.Done)
				case "migrated":
					types = append(types, model.Migrated)
				case "scheduled":
					types = append(types, model.Scheduled)
				case "event":
					types = append(types, model.Event)
				case "note":
					types = append(types, model.Note)
				case "important":
					types = append(types, model.HighlightImportant)
				case "inspiration":
					types = append(types, model.HighlightInspiration)
				}
			}
			u.state.SetTypeFilter(types)
			u.hideInput()
			u.refreshList()
			return nil
		}
		return event
	})
	u.showInput(field)
}

func (u *UI) showTagFilter() {
	// Comma-separated tags to include (any match)
	var def []string
	for t := range u.state.TagFilter {
		def = append(def, t)
	}
	field := tview.NewInputField().SetLabel("Tags filter (comma): ").SetText(strings.Join(def, ", ")).SetFieldWidth(60)
	field.SetBorder(false)
	styleInputField(field)
	field.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			u.hideInput()
			return nil
		case tcell.KeyEnter:
			raw := field.GetText()
			var tags []string
			for _, part := range strings.Split(raw, ",") {
				tt := strings.TrimSpace(part)
				if tt != "" {
					tags = append(tags, tt)
				}
			}
			u.state.SetTagFilter(tags)
			u.hideInput()
			u.refreshList()
			return nil
		}
		return event
	})
	u.showInput(field)
}

func (u *UI) showHelp() {
	lines := []string{
		"[red::b]BLT[-] — Help",
		"",
		"Movement:",
		"  j/k   PgUp/PgDn   g/G",
		"",
		"Scope:",
		"  1 Day   2 Week   3 Month",
		"  [ Prev  ] Next   d Jump to date   T Today",
		"",
		"Filters:",
		"  / Text  : Type (toggle)   F Tags",
		"",
		"Modify (Day only):",
		"  a Add   e Edit   x Delete   t Type   # Tags",
		"",
		"Actions:",
		"  c Complete (toggle Task/Done)",
		"  m Migrate (toggle on Migrated)",
		"  s Schedule (toggle on Scheduled)",
		"",
		"Close: Esc",
	}
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetText(strings.Join(lines, "\n")).
		SetTextAlign(tview.AlignLeft)
	// No border; use rules and padding for a clean look that fits small widths
	tv.SetBorder(false)
	tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			u.pages.RemovePage("help")
			u.app.SetFocus(u.list)
			return nil
		}
		// Swallow other keys so background doesn't react while help is open
		return nil
	})
	container := wrapWithRules(tv)
	overlay := pad(container, 1, 1)
	u.pages.AddPage("help", overlay, true, true)
	u.app.SetFocus(tv)
}

// center returns a centered primitive with a fixed size.
func center(w, h int, p tview.Primitive) tview.Primitive {
	return tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false). // top spacer
		AddItem(tview.NewFlex().
			AddItem(tview.NewBox(), 0, 1, false). // left spacer
			AddItem(p, w, 0, true).               // content with fixed width
			AddItem(tview.NewBox(), 0, 1, false), // right spacer
							h, 0, true). // middle row with fixed height
		AddItem(tview.NewBox(), 0, 1, false) // bottom spacer
}

func moveDown(l *tview.List) {
	idx := l.GetCurrentItem()
	if idx < l.GetItemCount()-1 {
		l.SetCurrentItem(idx + 1)
	}
}

func moveUp(l *tview.List) {
	idx := l.GetCurrentItem()
	if idx > 0 {
		l.SetCurrentItem(idx - 1)
	}
}

func moveHome(l *tview.List) { l.SetCurrentItem(0) }

func moveEnd(l *tview.List) {
	if c := l.GetItemCount(); c > 0 {
		l.SetCurrentItem(c - 1)
	}
}

func pageDown(l *tview.List) {
	_, _, _, h := l.GetRect()
	step := h - 3
	if step < 1 {
		step = 5
	}
	idx := l.GetCurrentItem()
	c := l.GetItemCount()
	idx += step
	if idx > c-1 {
		idx = c - 1
	}
	l.SetCurrentItem(idx)
}

func pageUp(l *tview.List) {
	_, _, _, h := l.GetRect()
	step := h - 3
	if step < 1 {
		step = 5
	}
	idx := l.GetCurrentItem()
	idx -= step
	if idx < 0 {
		idx = 0
	}
	l.SetCurrentItem(idx)
}

func formatBullet(b model.Bullet) string {
	prefix := "•"
	switch b.Type {
	case model.Done:
		prefix = "✓"
	case model.Migrated:
		prefix = "»"
	case model.Scheduled:
		prefix = "⧗"
	case model.Event:
		prefix = "◇"
	case model.Note:
		prefix = "–"
	case model.HighlightImportant:
		prefix = "!"
	case model.HighlightInspiration:
		prefix = "★"
	}
	tagStr := ""
	if len(b.Tags) > 0 {
		// Render tags as #tag format
		parts := make([]string, 0, len(b.Tags))
		for _, t := range b.Tags {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			if strings.HasPrefix(t, "#") {
				parts = append(parts, t)
			} else {
				parts = append(parts, "#"+t)
			}
		}
		if len(parts) > 0 {
			tagStr = "  [" + strings.Join(parts, " ") + "]"
		}
	}
	return prefix + " " + b.Text + tagStr
}

func (u *UI) formatEntry(e app.Entry) string {
	label := formatBullet(e.Item)
	if u.state.Period != model.PeriodDay {
		return e.Date.Format("2006-01-02") + "  " + label
	}
	return label
}

// contextControls builds the controls footer dynamically based on selection and scope.
func (u *UI) contextControls() string {
	// Base navigation/help always visible
	nav := "  [j/k] Move  [T] Today  [?] Help"
	// In non-day views, show default plus note handled by caller
	if u.state.Period != model.PeriodDay {
		return DefaultControls
	}
	// Day view: build modify keys based on selected item's type
	var parts []string
	parts = append(parts, "[a] Add", "[e] Edit", "[x] Delete", "[t] Type", "[#] Tags")
	// Resolve current selection
	idx := u.list.GetCurrentItem()
	vis := u.state.Visible()
	if idx >= 0 && idx < len(vis) {
		typ := vis[idx].Item.Type
		// Complete available for Task and Done (toggle), not for Migrated/Scheduled/others
		if typ == model.Task || typ == model.Done {
			if typ == model.Done {
				parts = append(parts, "[c] In progress")
			} else {
				parts = append(parts, "[c] Complete")
			}
		}
		// Migrate available for Task/Event and Migrated (for undo)
		if typ == model.Task || typ == model.Event || typ == model.Migrated {
			if typ == model.Migrated {
				parts = append(parts, "[m] Unmigrate")
			} else {
				parts = append(parts, "[m] Migrate")
			}
		}
		// Schedule available for Task/Event and Scheduled (for undo)
		if typ == model.Task || typ == model.Event || typ == model.Scheduled {
			if typ == model.Scheduled {
				parts = append(parts, "[s] Unschedule")
			} else {
				parts = append(parts, "[s] Schedule")
			}
		}
	} else {
		// No selection (empty state) — show minimal to encourage adding
		parts = []string{"[a] Add"}
	}
	return strings.Join(parts, "  ") + nav
}

// percentComplete computes percentage of completed actionable items (Task/Done) in the visible range.
// Returns -1 when there are no actionable items.
func (u *UI) percentComplete() int {
	vis := u.state.Visible()
	total := 0
	done := 0
	for _, e := range vis {
		switch e.Item.Type {
		case model.Task:
			total++
		case model.Done:
			total++
			done++
		}
	}
	if total == 0 {
		return -1
	}
	return int((float64(done) / float64(total)) * 100.0)
}

func (u *UI) showTypePicker() {
	idx := u.list.GetCurrentItem()
	vis := u.state.Visible()
	if idx < 0 || idx >= len(vis) {
		return
	}

	// Allowed semantic types only (exclude Done, Migrated, Scheduled)
	options := []struct {
		label string
		t     model.BulletType
	}{
		{"Task", model.Task},
		{"Event", model.Event},
		{"Note", model.Note},
		{"Important", model.HighlightImportant},
		{"Inspiration", model.HighlightInspiration},
	}

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)
	for _, opt := range options {
		typ := opt.t
		list.AddItem(opt.label, "", 0, func() {
			_ = u.state.ChangeTypeIndex(idx, typ)
			u.refreshList()
			u.pages.RemovePage("type-picker")
			u.inputActive = false
			u.app.SetFocus(u.list)
			u.updateStatus()
		})
	}
	// j/k navigation in addition to arrows, Esc to cancel
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			u.pages.RemovePage("type-picker")
			u.inputActive = false
			u.app.SetFocus(u.list)
			u.updateStatus()
			return nil
		}
		if ev.Key() == tcell.KeyRune {
			switch ev.Rune() {
			case 'j':
				moveDown(list)
				return nil
			case 'k':
				moveUp(list)
				return nil
			}
		}
		return ev
	})

	// Hints line shown within overlay so it remains visible
	hintText := "[j/k] Move   [enter] Select   [esc] Cancel"
	hints := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(false).
		SetText(hintText)
	hints.SetBorder(false)

	// Compose compact overlay with rules
	inner := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(list, len(options), 0, true).
		AddItem(hints, 1, 0, false)
	overlay := wrapWithRules(inner)

	// Center with responsive width and height (avoid fullscreen)
	height := len(options) + 3 // rules + list + hints
	if height < 6 {
		height = 6
	}
	desired := len(hintText) + 2
	for _, opt := range options {
		if l := len(opt.label) + 4; l > desired {
			desired = l
		}
	}
	maxW := u.centerWidth
	if desired < 28 {
		desired = 28
	}
	if desired > maxW {
		desired = maxW
	}
	u.pages.AddPage("type-picker", center(desired, height, overlay), true, true)
	u.inputActive = true
	u.app.SetFocus(list)
	u.updateStatus()
}

func (u *UI) showTagsDialog() {
	idx := u.list.GetCurrentItem()
	vis := u.state.Visible()
	if idx < 0 || idx >= len(vis) {
		return
	}
	curr := vis[idx].Item
	def := strings.Join(curr.Tags, ", ")
	field := tview.NewInputField().SetLabel("Tags (comma): ").SetText(def).SetFieldWidth(60)
	field.SetBorder(false)
	styleInputField(field)
	field.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			u.hideInput()
			return nil
		case tcell.KeyEnter:
			raw := field.GetText()
			var tags []string
			for _, part := range strings.Split(raw, ",") {
				t := strings.TrimSpace(part)
				if t != "" {
					// Normalize: store without leading '#'
					tags = append(tags, strings.TrimPrefix(t, "#"))
				}
			}
			_ = u.state.UpdateTagsIndex(idx, tags)
			u.hideInput()
			u.refreshList()
			return nil
		}
		return event
	})
	u.showInput(field)
}

// showError displays a blocking modal with an OK button.
func (u *UI) showError(msg string) {
	// Inline ephemeral error in controls footer
	u.controls.SetText("[red]" + msg + "[-]  " + u.contextControls() + u.filtersSummary())
}

// Inline input helpers
func (u *UI) showInput(p tview.Primitive) {
	if u.inputActive && u.inputPrimitive != nil {
		u.grid.RemoveItem(u.inputPrimitive)
	}
	// Any previous confirmation state is cleared when showing a real input
	u.confirmCallback = nil
	u.promptMessage = ""
	// Wrap the input with top/bottom rules to avoid thick/double borders
	container := wrapWithRules(p)
	u.inputPrimitive = container
	u.inputActive = true
	// Resize grid to show input row and attach primitive
	u.grid.SetRows(1, 1, 0, 3, 3)
	u.grid.AddItem(container, 3, 1, 1, 1, 0, 0, true)
	u.app.SetFocus(p)
	u.updateStatus()
}

func (u *UI) hideInput() {
	if u.inputPrimitive != nil {
		u.grid.RemoveItem(u.inputPrimitive)
	}
	u.inputActive = false
	u.inputPrimitive = nil
	u.confirmCallback = nil
	u.promptMessage = ""
	// Hide input row
	u.grid.SetRows(1, 1, 0, 0, 3)
	u.app.SetFocus(u.list)
	u.updateStatus()
}

// styleInputField applies consistent styling to input fields
func styleInputField(f *tview.InputField) {
	f.SetBackgroundColor(tcell.ColorDefault)
	f.SetFieldBackgroundColor(tcell.ColorDefault)
	// Ensure single-line (non-bold) border appearance
	// If available in this tview version, remove bold border attributes.
	// This call is safe even if it no-ops.
	f.SetBorderAttributes(tcell.AttrNone)
}

// wrapWithRules surrounds a primitive with a simple top and bottom horizontal rule.
func wrapWithRules(p tview.Primitive) tview.Primitive {
	top := tview.NewTextView().SetDynamicColors(true)
	bottom := tview.NewTextView().SetDynamicColors(true)
	line := "[green]" + strings.Repeat("─", 200)
	top.SetText(line)
	bottom.SetText(line)
	return tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(top, 1, 0, false).
		// Let content expand to fill available vertical space
		AddItem(p, 0, 1, true).
		AddItem(bottom, 1, 0, false)
}

// pad adds horizontal and vertical padding around a primitive while letting it fill remaining space.
func pad(p tview.Primitive, hpad, vpad int) tview.Primitive {
	return tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), vpad, 0, false).
		AddItem(tview.NewFlex().
			AddItem(tview.NewBox(), hpad, 0, false).
			AddItem(p, 0, 1, true).
			AddItem(tview.NewBox(), hpad, 0, false),
			0, 1, true).
		AddItem(tview.NewBox(), vpad, 0, false)
}

// showToast displays a non-blocking message that auto-dismisses.
// toasts removed per UX decision; rely on immediate list updates and footer note
