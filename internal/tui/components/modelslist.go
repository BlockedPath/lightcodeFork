package components

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/config"
)

const maxModelsListHeight = 10

type modelItem config.ResModel

func (i modelItem) FilterValue() string { return i.Model }

type modelItemDelegate struct {
	styles *styles
}

func (d modelItemDelegate) Height() int                             { return 1 }
func (d modelItemDelegate) Spacing() int                            { return 0 }
func (d modelItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d modelItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(modelItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s", i.Model)

	fn := d.styles.item.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return d.styles.selectedItem.Render(lipgloss.NewStyle().Foreground(lipgloss.BrightCyan).Render("→ " + strings.Join(s, " ")))
		}
	}

	fmt.Fprint(w, fn(str))
}

type ModelModelsList struct {
	list     list.Model
	allItems []list.Item
	styles   styles
}

func initialModelsList() ModelModelsList {
	const defaultWidth = 20

	l := list.New([]list.Item{}, modelItemDelegate{}, defaultWidth, 1)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowTitle(false)
	l.SetShowFilter(false)
	l.SetDelegate(list.DefaultDelegate{})

	m := ModelModelsList{list: l, allItems: []list.Item{}}
	m.updateStyles(true)
	return m
}

func (m *ModelModelsList) Filter(term string) {
	if term == "" {
		m.list.SetItems(m.allItems)
		m.list.SetHeight(modelsListHeight(len(m.allItems)))
		return
	}
	var filtered []list.Item
	for _, i := range m.allItems {
		if strings.Contains(strings.ToLower(i.FilterValue()), strings.ToLower(term)) {
			filtered = append(filtered, i)
		}
	}
	m.list.SetItems(filtered)
	m.list.SetHeight(modelsListHeight(len(filtered)))
}

func (m *ModelModelsList) Refresh(items []config.ResModel) {
	listItems := make([]list.Item, len(items))
	for i, model := range items {
		listItems[i] = modelItem(model)
	}
	m.allItems = listItems
	m.list.SetItems(listItems)
	m.list.SetHeight(modelsListHeight(len(listItems)))
}

func (m *ModelModelsList) updateStyles(isDark bool) {
	m.styles = newStyles(isDark)
	m.list.Styles.Title = m.styles.title
	m.list.Styles.PaginationStyle = m.styles.pagination
	m.list.Styles.HelpStyle = m.styles.help
	m.list.SetDelegate(modelItemDelegate{styles: &m.styles})
}

func (m ModelModelsList) Init() tea.Cmd {
	return nil
}

func (m ModelModelsList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "down":
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m ModelModelsList) View() tea.View {
	return tea.NewView("\n" + m.list.View())
}

func (m ModelModelsList) StringView() string {
	return m.list.View()
}

func (m ModelModelsList) Current() config.ResModel {
	selected := m.list.SelectedItem()
	if selected == nil {
		return config.ResModel{}
	}
	it, ok := selected.(modelItem)
	if !ok {
		return config.ResModel{}
	}
	return config.ResModel(it)
}

func (m ModelModelsList) Height() int {
	return m.list.Height()
}

func modelsListHeight(items int) int {
	if items <= 0 {
		return 1
	}
	if items > maxModelsListHeight {
		return maxModelsListHeight
	}
	return items
}

func LaunchModelsList() ModelModelsList {
	return initialModelsList()
}
