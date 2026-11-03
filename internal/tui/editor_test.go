package tui

import (
	"fmt"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/jomei/notionapi"
)

func TestEditorFormIgnoresKeysUntilReady(t *testing.T) {
	keys := []tea.KeyPressMsg{
		{Code: tea.KeyUp}, {Code: tea.KeyDown}, {Code: tea.KeyTab}, {Code: tea.KeyEnter},
		{Code: 'n', Mod: tea.ModCtrl}, {Code: 'p', Mod: tea.ModCtrl},
		{Code: 's', Mod: tea.ModCtrl}, {Code: 'x', Text: "x"},
	}
	for _, mode := range []editorMode{modeCreate, modeEdit} {
		for _, state := range []struct {
			name               string
			loading, hasInputs bool
		}{
			{name: "loading", loading: true},
			{name: "no editable properties"},
			{name: "saving", loading: true, hasInputs: true},
		} {
			for _, key := range keys {
				t.Run(fmt.Sprintf("mode=%d/%s/%s", mode, state.name, key.String()), func(t *testing.T) {
					m := NewEditorModel(nil)
					m.mode = mode
					m.initPropertyInputs(&notionapi.Database{Properties: notionapi.PropertyConfigs{}}, nil)
					if state.hasInputs {
						m.initPropertyInputs(&notionapi.Database{Properties: notionapi.PropertyConfigs{
							"Title": &notionapi.TitlePropertyConfig{Type: notionapi.PropertyConfigTypeTitle},
						}}, nil)
						m.inputs[0].SetValue("original")
					}
					m.loading = state.loading
					m, cmd := m.Update(key)
					if cmd != nil || m.focusedIdx != 0 || m.loading != state.loading {
						t.Fatal("unavailable form handled input or scheduled a save")
					}
					if state.hasInputs && m.inputs[0].Value() != "original" {
						t.Fatal("input changed while saving")
					}
				})
			}
		}
	}
}

func TestEditorUnavailableFormCanCancel(t *testing.T) {
	for _, mode := range []editorMode{modeCreate, modeEdit} {
		for _, loading := range []bool{true, false} {
			m := NewEditorModel(nil)
			m.mode, m.loading = mode, loading
			_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			if cmd == nil {
				t.Fatal("Escape did not schedule cancellation")
			}
			if _, ok := cmd().(CancelEditMsg); !ok {
				t.Fatal("Escape did not cancel editing")
			}
		}
	}
}

func TestEditorFormBecomesUsableAfterLoading(t *testing.T) {
	for _, mode := range []editorMode{modeCreate, modeEdit} {
		m := NewEditorModel(nil)
		m.mode, m.loading = mode, true
		m, _ = m.Update(&notionapi.Database{Properties: notionapi.PropertyConfigs{
			"Title": &notionapi.TitlePropertyConfig{Type: notionapi.PropertyConfigTypeTitle},
			"Notes": &notionapi.RichTextPropertyConfig{Type: notionapi.PropertyConfigTypeRichText},
		}})
		if m.loading || len(m.inputs) != 2 {
			t.Fatal("metadata did not initialize form")
		}
		m, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
		if m.inputs[0].Value() != "x" {
			t.Fatal("loaded form did not accept typing")
		}
		m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
		if m.focusedIdx != 1 || !m.inputs[1].Focused() {
			t.Fatal("ctrl+n did not focus next field")
		}
		m, _ = m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
		if m.focusedIdx != 0 || !m.inputs[0].Focused() {
			t.Fatal("ctrl+p did not focus previous field")
		}
		m, cmd := m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
		if !m.loading || cmd == nil {
			t.Fatal("ready form did not schedule save")
		}
		// Do not run the save command: this test must not contact Notion.
	}
}

func TestEditablePropertyKeys(t *testing.T) {
	configs := map[string]notionapi.PropertyConfig{
		"URL":    &notionapi.URLPropertyConfig{Type: notionapi.PropertyConfigTypeURL},
		"Title":  &notionapi.TitlePropertyConfig{Type: notionapi.PropertyConfigTypeTitle},
		"Status": &notionapi.SelectPropertyConfig{Type: notionapi.PropertyConfigTypeSelect},
		"Created": &notionapi.CreatedTimePropertyConfig{
			ID:   "created",
			Type: notionapi.PropertyConfigCreatedTime,
		},
		"Notes": &notionapi.RichTextPropertyConfig{Type: notionapi.PropertyConfigTypeRichText},
	}

	got := editablePropertyKeys(configs)
	want := []string{"Title", "Notes", "Status", "URL"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("editablePropertyKeys() = %v, want %v", got, want)
	}
}

func TestInputProperty(t *testing.T) {
	tests := []struct {
		name   string
		config notionapi.PropertyConfig
		value  string
		wantOK bool
		assert func(t *testing.T, prop notionapi.Property)
	}{
		{
			name:   "title",
			config: &notionapi.TitlePropertyConfig{Type: notionapi.PropertyConfigTypeTitle},
			value:  "Task",
			wantOK: true,
			assert: func(t *testing.T, prop notionapi.Property) {
				p, ok := prop.(notionapi.TitleProperty)
				if !ok {
					t.Fatalf("prop type = %T, want notionapi.TitleProperty", prop)
				}
				if got := p.Title[0].Text.Content; got != "Task" {
					t.Fatalf("title content = %q, want %q", got, "Task")
				}
			},
		},
		{
			name:   "invalid number",
			config: &notionapi.NumberPropertyConfig{Type: notionapi.PropertyConfigTypeNumber},
			value:  "not-a-number",
			wantOK: false,
		},
		{
			name:   "multi select trims values",
			config: &notionapi.MultiSelectPropertyConfig{Type: notionapi.PropertyConfigTypeMultiSelect},
			value:  "bug,  feature, ",
			wantOK: true,
			assert: func(t *testing.T, prop notionapi.Property) {
				p, ok := prop.(notionapi.MultiSelectProperty)
				if !ok {
					t.Fatalf("prop type = %T, want notionapi.MultiSelectProperty", prop)
				}
				got := []string{}
				for _, option := range p.MultiSelect {
					got = append(got, option.Name)
				}
				want := []string{"bug", "feature"}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("multi select options = %v, want %v", got, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop, ok := inputProperty(tt.config, tt.value)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.assert != nil {
				tt.assert(t, prop)
			}
		})
	}
}
