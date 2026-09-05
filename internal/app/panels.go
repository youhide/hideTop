package app

// panelName identifies a metric panel for visibility toggling and layout.
type panelName string

const (
	panelCPU  panelName = "cpu"
	panelGPU  panelName = "gpu"
	panelMem  panelName = "mem"
	panelTemp panelName = "temp"
	panelNet  panelName = "net"
	panelDisk panelName = "disk"
)

// panelOrder is the order panels are packed into columns, and the order the
// number keys 1-6 address them.
var panelOrder = []panelName{panelCPU, panelGPU, panelMem, panelTemp, panelNet, panelDisk}

// panelVisible reports whether a panel should be rendered. Hiding is opt-in
// per panel and persisted in the config file.
func (m Model) panelVisible(name panelName) bool {
	return !m.hiddenPanels[string(name)]
}

// togglePanel flips a panel's visibility and marks the config dirty so the
// choice is written on quit.
func (m *Model) togglePanel(name panelName) {
	if m.hiddenPanels == nil {
		m.hiddenPanels = make(map[string]bool, len(panelOrder))
	}
	if m.hiddenPanels[string(name)] {
		delete(m.hiddenPanels, string(name))
	} else {
		m.hiddenPanels[string(name)] = true
	}
	m.panelsChanged = true
}

// hiddenPanelList returns the hidden panel names in panelOrder order, for
// persisting to the config file.
func (m Model) hiddenPanelList() []string {
	var out []string
	for _, n := range panelOrder {
		if m.hiddenPanels[string(n)] {
			out = append(out, string(n))
		}
	}
	return out
}
