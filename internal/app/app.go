package app

import (
	"strings"
	"time"

	"github.com/rdo34/blt/internal/model"
	"github.com/rdo34/blt/internal/store"
)

// Entry represents a bullet and its owning date, enabling range views.
type Entry struct {
	Date time.Time
	Item model.Bullet
}

// App holds application state and provides methods to load/update bullets.
type App struct {
	Store       store.Store
	CurrentDate time.Time
	Period      model.Period

	Items []Entry

	// Filters
	TextFilter string
	TypeFilter map[model.BulletType]bool
	TagFilter  map[string]bool // normalized to no leading '#'
}

func New(s store.Store) *App {
	return &App{Store: s, Period: model.PeriodDay, TypeFilter: map[model.BulletType]bool{}, TagFilter: map[string]bool{}}
}

// LoadDay loads all bullets for a given date into state.
func (a *App) LoadDay(date time.Time) error { a.CurrentDate = date; a.SavePrefs(); return a.Refresh() }

// Refresh reloads items for the current period and date, applying no filters.
func (a *App) Refresh() error {
	rng := a.dateRange()
	var entries []Entry
	for d := rng.Start; !d.After(rng.End); d = d.Add(24 * time.Hour) {
		items, err := a.Store.LoadDay(d)
		if err != nil {
			return err
		}
		for _, it := range items {
			entries = append(entries, Entry{Date: dateOnly(d), Item: it})
		}
	}
	a.Items = entries
	return nil
}

// Visible returns items matching current filters.
func (a *App) Visible() []Entry {
	out := make([]Entry, 0, len(a.Items))
	for _, e := range a.Items {
		if a.match(e) {
			out = append(out, e)
		}
	}
	return out
}

func (a *App) match(e Entry) bool {
	if a.TextFilter != "" {
		if !strings.Contains(strings.ToLower(e.Item.Text), strings.ToLower(a.TextFilter)) {
			return false
		}
	}
	if len(a.TypeFilter) > 0 {
		if !a.TypeFilter[e.Item.Type] {
			return false
		}
	}
	if len(a.TagFilter) > 0 {
		ok := false
		for _, t := range e.Item.Tags {
			if a.TagFilter[strings.TrimPrefix(strings.ToLower(t), "#")] {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// Period controls
func (a *App) SetPeriod(p model.Period) error { a.Period = p; a.SavePrefs(); return a.Refresh() }
func (a *App) NextPeriod() error              { a.CurrentDate = a.shift(1); a.SavePrefs(); return a.Refresh() }
func (a *App) PrevPeriod() error              { a.CurrentDate = a.shift(-1); a.SavePrefs(); return a.Refresh() }
func (a *App) JumpToDate(d time.Time) error {
	a.CurrentDate = dateOnly(d)
	a.SavePrefs()
	return a.Refresh()
}

func (a *App) shift(delta int) time.Time {
	switch a.Period {
	case model.PeriodWeek:
		return a.CurrentDate.AddDate(0, 0, 7*delta)
	case model.PeriodMonth:
		return a.CurrentDate.AddDate(0, delta, 0)
	default:
		return a.CurrentDate.AddDate(0, 0, delta)
	}
}

func (a *App) dateRange() model.DateRange {
	d := dateOnly(a.CurrentDate)
	switch a.Period {
	case model.PeriodWeek:
		// Monday-start week
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

func dateOnly(t time.Time) time.Time {
	y, m, day := t.Date()
	return time.Date(y, m, day, 0, 0, 0, 0, t.Location())
}

// Add creates a new default Task bullet for the current date.
func (a *App) Add(text string) (model.Bullet, error) {
	b := model.Bullet{Text: text, Type: model.Task, CreatedAt: time.Now()}
	if err := a.Store.Append(a.CurrentDate, b); err != nil {
		return model.Bullet{}, err
	}
	// Reload range to pick up ID.
	if err := a.Refresh(); err != nil {
		return model.Bullet{}, err
	}
	return b, nil
}

// UpdateText updates the text of an item by index for the current date.
func (a *App) UpdateText(index int, text string) error {
	vis := a.Visible()
	if index < 0 || index >= len(vis) {
		return nil
	}
	e := vis[index]
	it := e.Item
	it.Text = text
	if err := a.Store.Update(e.Date, it); err != nil {
		return err
	}
	return a.Refresh()
}

// DeleteIndex removes an item by index from the current date.
func (a *App) DeleteIndex(index int) error {
	vis := a.Visible()
	if index < 0 || index >= len(vis) {
		return nil
	}
	e := vis[index]
	if err := a.Store.Delete(e.Date, e.Item.ID); err != nil {
		return err
	}
	return a.Refresh()
}

// CompleteIndex marks the item as done for the current date.
func (a *App) CompleteIndex(index int) error {
	vis := a.Visible()
	if index < 0 || index >= len(vis) {
		return nil
	}
	e := vis[index]
	it := e.Item
	// Do not allow completing migrated or scheduled items
	if it.Type == model.Migrated || it.Type == model.Scheduled {
		return nil
	}
	// Toggle: Task <-> Done; ignore other types
	if it.Type == model.Task {
		now := time.Now()
		it.Type = model.Done
		it.CompletedAt = &now
	} else if it.Type == model.Done {
		it.Type = model.Task
		it.CompletedAt = nil
	} else {
		return nil
	}
	if err := a.Store.Update(e.Date, it); err != nil {
		return err
	}
	return a.Refresh()
}

// MigrateIndex moves the item to the next day and removes it from today.
func (a *App) MigrateIndex(index int) error {
	vis := a.Visible()
	if index < 0 || index >= len(vis) {
		return nil
	}
	e := vis[index]
	orig := e.Item
	// If already migrated, undo the migration.
	if orig.Type == model.Migrated {
		// Determine target date: prefer ScheduledFor if set, else next day
		target := e.Date.Add(24 * time.Hour)
		if orig.ScheduledFor != nil {
			target = dateOnly(*orig.ScheduledFor)
		}
		// Find the migrated clone on target day and delete it, restoring original type
		items, err := a.Store.LoadDay(target)
		if err != nil {
			return err
		}
		var cloneID string
		var restoreType model.BulletType = model.Task
		for _, it := range items {
			if strings.EqualFold(it.Text, orig.Text) {
				// Only consider likely original types (Task/Event)
				if it.Type == model.Task || it.Type == model.Event {
					cloneID = it.ID
					restoreType = it.Type
					break
				}
			}
		}
		// Update current-day item back to the original type
		orig.Type = restoreType
		orig.CompletedAt = nil
		orig.ScheduledFor = nil
		if err := a.Store.Update(e.Date, orig); err != nil {
			return err
		}
		// Delete the clone if we found one
		if cloneID != "" {
			if err := a.Store.Delete(target, cloneID); err != nil {
				return err
			}
		}
		return a.Refresh()
	}
	// Only migrate tasks or events
	if orig.Type != model.Task && orig.Type != model.Event {
		return nil
	}
	// 1) Mark current-day item as Migrated (retain it on the current date)
	marked := orig
	marked.Type = model.Migrated
	marked.CompletedAt = nil
	next := e.Date.Add(24 * time.Hour)
	// Store target date to enable undo
	marked.ScheduledFor = &next
	if err := a.Store.Update(e.Date, marked); err != nil {
		return err
	}
	// 2) Add a copy on the next day with the original type
	clone := orig
	clone.ID = ""          // new identity on target day
	clone.Type = orig.Type // keep original style
	clone.CompletedAt = nil
	clone.ScheduledFor = nil
	clone.CreatedAt = time.Time{} // let Append set now
	if err := a.Store.Append(next, clone); err != nil {
		return err
	}
	return a.Refresh()
}

// ScheduleIndex moves the item to a specific date and marks as scheduled.
func (a *App) ScheduleIndex(index int, date time.Time) error {
	vis := a.Visible()
	if index < 0 || index >= len(vis) {
		return nil
	}
	e := vis[index]
	orig := e.Item
	// If already scheduled, undo the scheduling.
	if orig.Type == model.Scheduled {
		if orig.ScheduledFor == nil {
			return nil
		}
		target := dateOnly(*orig.ScheduledFor)
		items, err := a.Store.LoadDay(target)
		if err != nil {
			return err
		}
		var cloneID string
		var restoreType model.BulletType = model.Task
		for _, it := range items {
			if strings.EqualFold(it.Text, orig.Text) {
				if it.Type == model.Task || it.Type == model.Event {
					cloneID = it.ID
					restoreType = it.Type
					break
				}
			}
		}
		// Restore current-day item
		orig.Type = restoreType
		orig.ScheduledFor = nil
		orig.CompletedAt = nil
		if err := a.Store.Update(e.Date, orig); err != nil {
			return err
		}
		if cloneID != "" {
			if err := a.Store.Delete(target, cloneID); err != nil {
				return err
			}
		}
		return a.Refresh()
	}
	// Only schedule tasks or events
	if orig.Type != model.Task && orig.Type != model.Event {
		return nil
	}
	// 1) Mark current-day item as Scheduled and keep it
	marked := orig
	marked.Type = model.Scheduled
	marked.ScheduledFor = &date
	marked.CompletedAt = nil
	if err := a.Store.Update(e.Date, marked); err != nil {
		return err
	}
	// 2) Add a copy on the scheduled day with the original style
	clone := orig
	clone.ID = ""
	clone.Type = orig.Type
	clone.ScheduledFor = nil
	clone.CompletedAt = nil
	clone.CreatedAt = time.Time{}
	if err := a.Store.Append(dateOnly(date), clone); err != nil {
		return err
	}
	return a.Refresh()
}

// ChangeTypeIndex updates the type and adjusts related fields.
func (a *App) ChangeTypeIndex(index int, t model.BulletType) error {
	vis := a.Visible()
	if index < 0 || index >= len(vis) {
		return nil
	}
	e := vis[index]
	it := e.Item
	it.Type = t
	// Normalize related fields
	if t == model.Done {
		now := time.Now()
		it.CompletedAt = &now
	} else {
		it.CompletedAt = nil
	}
	if t != model.Scheduled {
		it.ScheduledFor = nil
	}
	if err := a.Store.Update(e.Date, it); err != nil {
		return err
	}
	return a.Refresh()
}

// UpdateTagsIndex replaces the tags list for an item.
func (a *App) UpdateTagsIndex(index int, tags []string) error {
	vis := a.Visible()
	if index < 0 || index >= len(vis) {
		return nil
	}
	e := vis[index]
	it := e.Item
	it.Tags = tags
	if err := a.Store.Update(e.Date, it); err != nil {
		return err
	}
	return a.Refresh()
}

// Filters API
func (a *App) SetTextFilter(q string) { a.TextFilter = strings.TrimSpace(q); a.SavePrefs() }
func (a *App) SetTypeFilter(types []model.BulletType) {
	a.TypeFilter = map[model.BulletType]bool{}
	for _, t := range types {
		a.TypeFilter[t] = true
	}
	a.SavePrefs()
}
func (a *App) SetTagFilter(tags []string) {
	a.TagFilter = map[string]bool{}
	for _, t := range tags {
		a.TagFilter[strings.ToLower(strings.TrimPrefix(strings.TrimSpace(t), "#"))] = true
	}
	a.SavePrefs()
}
func (a *App) ClearFilters() {
	a.TextFilter = ""
	a.TypeFilter = map[model.BulletType]bool{}
	a.TagFilter = map[string]bool{}
	a.SavePrefs()
}

// SavePrefs persists the current period, filters, and date.
func (a *App) SavePrefs() {
	// Load existing prefs to preserve unrelated fields (e.g., UI settings)
	p, _ := store.LoadPreferences()
	// Update controlled fields
	var types []model.BulletType
	for t, ok := range a.TypeFilter {
		if ok {
			types = append(types, t)
		}
	}
	var tags []string
	for t, ok := range a.TagFilter {
		if ok {
			tags = append(tags, t)
		}
	}
	p.Period = a.Period
	p.TextFilter = a.TextFilter
	p.Types = types
	p.Tags = tags
	p.LastDate = a.CurrentDate.Format("2006-01-02")
	_ = store.SavePreferences(p)
}

// --- Day-index operations for CLI (ignore filters/range) ---

// DeleteDayIndex removes an item by zero-based index within a specific day.
func (a *App) DeleteDayIndex(date time.Time, index int) error {
	items, err := a.Store.LoadDay(dateOnly(date))
	if err != nil {
		return err
	}
	if index < 0 || index >= len(items) {
		return nil
	}
	return a.Store.Delete(dateOnly(date), items[index].ID)
}

// UpdateDayIndexText updates text by zero-based index within a day.
func (a *App) UpdateDayIndexText(date time.Time, index int, text string) error {
	items, err := a.Store.LoadDay(dateOnly(date))
	if err != nil {
		return err
	}
	if index < 0 || index >= len(items) {
		return nil
	}
	it := items[index]
	it.Text = text
	return a.Store.Update(dateOnly(date), it)
}

// CompleteDayIndex toggles Task<->Done for a day item; no-op for other types.
func (a *App) CompleteDayIndex(date time.Time, index int) error {
	d := dateOnly(date)
	items, err := a.Store.LoadDay(d)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(items) {
		return nil
	}
	it := items[index]
	if it.Type == model.Migrated || it.Type == model.Scheduled {
		return nil
	}
	if it.Type == model.Task {
		now := time.Now()
		it.Type = model.Done
		it.CompletedAt = &now
	} else if it.Type == model.Done {
		it.Type = model.Task
		it.CompletedAt = nil
	} else {
		return nil
	}
	return a.Store.Update(d, it)
}

// MigrateDayIndex toggles migration for Task/Event and Migrated.
func (a *App) MigrateDayIndex(date time.Time, index int) error {
	d := dateOnly(date)
	items, err := a.Store.LoadDay(d)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(items) {
		return nil
	}
	orig := items[index]
	if orig.Type == model.Migrated {
		target := d.AddDate(0, 0, 1)
		if orig.ScheduledFor != nil {
			target = dateOnly(*orig.ScheduledFor)
		}
		titems, err := a.Store.LoadDay(target)
		if err != nil {
			return err
		}
		var cloneID string
		restoreType := model.Task
		for _, it := range titems {
			if strings.EqualFold(it.Text, orig.Text) && (it.Type == model.Task || it.Type == model.Event) {
				cloneID = it.ID
				restoreType = it.Type
				break
			}
		}
		orig.Type = restoreType
		orig.CompletedAt = nil
		orig.ScheduledFor = nil
		if err := a.Store.Update(d, orig); err != nil {
			return err
		}
		if cloneID != "" {
			return a.Store.Delete(target, cloneID)
		}
		return nil
	}
	if orig.Type != model.Task && orig.Type != model.Event {
		return nil
	}
	next := d.AddDate(0, 0, 1)
	marked := orig
	marked.Type = model.Migrated
	marked.CompletedAt = nil
	marked.ScheduledFor = &next
	if err := a.Store.Update(d, marked); err != nil {
		return err
	}
	clone := orig
	clone.ID = ""
	clone.CompletedAt = nil
	clone.ScheduledFor = nil
	clone.CreatedAt = time.Time{}
	return a.Store.Append(next, clone)
}

// ScheduleDayIndex toggles scheduling to a date or unschedules.
func (a *App) ScheduleDayIndex(date time.Time, index int, target time.Time) error {
	d := dateOnly(date)
	items, err := a.Store.LoadDay(d)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(items) {
		return nil
	}
	orig := items[index]
	if orig.Type == model.Scheduled {
		if orig.ScheduledFor == nil {
			return nil
		}
		targetDay := dateOnly(*orig.ScheduledFor)
		titems, err := a.Store.LoadDay(targetDay)
		if err != nil {
			return err
		}
		var cloneID string
		restoreType := model.Task
		for _, it := range titems {
			if strings.EqualFold(it.Text, orig.Text) && (it.Type == model.Task || it.Type == model.Event) {
				cloneID = it.ID
				restoreType = it.Type
				break
			}
		}
		orig.Type = restoreType
		orig.CompletedAt = nil
		orig.ScheduledFor = nil
		if err := a.Store.Update(d, orig); err != nil {
			return err
		}
		if cloneID != "" {
			return a.Store.Delete(targetDay, cloneID)
		}
		return nil
	}
	if orig.Type != model.Task && orig.Type != model.Event {
		return nil
	}
	tgt := dateOnly(target)
	marked := orig
	marked.Type = model.Scheduled
	marked.CompletedAt = nil
	marked.ScheduledFor = &tgt
	if err := a.Store.Update(d, marked); err != nil {
		return err
	}
	clone := orig
	clone.ID = ""
	clone.CompletedAt = nil
	clone.ScheduledFor = nil
	clone.CreatedAt = time.Time{}
	return a.Store.Append(tgt, clone)
}
