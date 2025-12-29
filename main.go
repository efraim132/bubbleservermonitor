package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type mode int

const (
	modeViewing mode = iota
	modeAdding
)

const pingInterval = 5 // seconds

const pingTimeout = 2 // seconds

const maxPingSince = 9999 // seconds

const appTitle = "Server Monitor"

const (
	statusOnline  = "●"
	statusOffline = "○"
)

type server struct {
	Name             string
	IP               string
	Port             int
	pingSinceSeconds int64
}

type model struct {
	addServerForm *huh.Form
	serverList    table.Model
	servers       []server
	focusIndex    int
	mode          mode
	// Form input fields
	formName string
	formIP   string
	formPort string
	// Styles
	onlineStyle  lipgloss.Style
	offlineStyle lipgloss.Style
	tableStyle   lipgloss.Style
}

type pingResultMsg struct {
	serverIndex int
	online      bool
}

type tickMsg struct{}

func initialModel() model {
	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "IP Address", Width: 15},
		{Title: "Port", Width: 6},
		{Title: "Status", Width: 8},
		{Title: "Ping Since (s)", Width: 15},
	}

	table := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(false),
		table.WithHeight(10),
		table.WithStyles(table.Styles{
			Header: lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("12")),
		}),
	)

	initialmodel := model{
		serverList:   table,
		servers:      []server{},
		focusIndex:   0,
		mode:         modeViewing,
		onlineStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("10")), // Green
		offlineStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("9")),  // Red
		tableStyle:   lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("13")),
	}

	initialmodel.addServerForm = initialmodel.generateAddForm()

	return initialmodel
}

func (m *model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Duration(pingInterval)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg.(type) {
	case tickMsg:
		// Start pinging all servers
		cmds := []tea.Cmd{}
		for i := range m.servers {
			index := i // Capture loop variable
			srv := m.servers[i]
			cmd := func() tea.Msg {
				online, _ := pingServer(&srv)
				return pingResultMsg{
					serverIndex: index,
					online:      online,
				}
			}
			cmds = append(cmds, cmd)
		}
		// Schedule next tick
		cmds = append(cmds, tick())
		return m, tea.Batch(cmds...)

	case pingResultMsg:
		result := msg.(pingResultMsg)
		if result.serverIndex < len(m.servers) {
			if result.online {
				m.servers[result.serverIndex].pingSinceSeconds = 0
			} else {
				if m.servers[result.serverIndex].pingSinceSeconds < maxPingSince {
					m.servers[result.serverIndex].pingSinceSeconds += pingInterval
				}
			}
		}
		return m, nil
	}

	switch m.mode {
	case modeViewing:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "a":
				m.mode = modeAdding
				m.formName = ""
				m.formIP = ""
				m.formPort = ""
				m.addServerForm = m.generateAddForm()
				return m, m.addServerForm.Init()
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}
	case modeAdding:
		form, cmd := m.addServerForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.addServerForm = f
		}

		if m.addServerForm.State == huh.StateCompleted {
			// Parse port from string
			port := 0
			fmt.Sscanf(m.formPort, "%d", &port)

			m.servers = append(m.servers, server{
				Name: m.formName,
				IP:   m.formIP,
				Port: port,
			})
			log.Printf("Added server: %s (%s:%d)", m.formName, m.formIP, port)
			m.addServerForm = m.generateAddForm()
			m.mode = modeViewing
		}
		return m, cmd
	}

	return m, nil
}

func (m *model) View() string {
	switch m.mode {
	case modeViewing:
		rows := []table.Row{}
		for _, srv := range m.servers {
			status := statusOnline
			var styledStatus string
			if srv.pingSinceSeconds > 0 {
				status = statusOffline
				styledStatus = m.offlineStyle.Render(status)
			} else {
				styledStatus = m.onlineStyle.Render(status)
			}
			pingSince := "0"
			if srv.pingSinceSeconds > 0 {
				pingSince = fmt.Sprintf("%d", srv.pingSinceSeconds)
			}
			rows = append(rows, table.Row{
				srv.Name,
				srv.IP,
				fmt.Sprintf("%d", srv.Port),
				styledStatus,
				pingSince,
			})
		}
		m.serverList.SetRows(rows)
		return appTitle + "\n\n" + m.tableStyle.Render(m.serverList.View()) + "\n\nPress 'a' to add a server, 'q' to quit."
	case modeAdding:
		return appTitle + "\n\n" + m.addServerForm.View() + "\n\nPress 'Enter' to submit, 'Esc' to cancel."
	}
	return ""
}

func (m *model) generateAddForm() *huh.Form {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Value(&m.formName),
			huh.NewInput().
				Title("IP Address").
				Value(&m.formIP),
			huh.NewInput().
				Title("Port").
				Value(&m.formPort),
		),
	)
	return form
}

func pingServer(srv *server) (bool, error) {
	// ping the server at srv.IP:srv.Port
	// return true if online, false if offline

	// net.JoinHostPort combines host and port into "host:port" format
	address := net.JoinHostPort(srv.IP, fmt.Sprintf("%d", srv.Port))
	timeout := time.Duration(pingTimeout) * time.Second

	// net.DialTimeout tries to connect with a timeout
	// "tcp" means we're using TCP protocol
	// If connection succeeds, conn will contain the connection object
	conn, err := net.DialTimeout("tcp", address, timeout)

	// If there was an error connecting, return false and the error
	if err != nil {
		return false, err
	}

	// defer means "run this when the function exits"
	// We need to close the connection to free up resources
	defer conn.Close()

	// If we got here, connection succeeded!
	return true, nil
}

func main() {
	logFile, err := os.OpenFile("debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close() // Now defer works correctly
	}
	m := initialModel()
	p := tea.NewProgram(&m)
	if err := p.Start(); err != nil {
		fmt.Println("Error starting program:", err)
		os.Exit(1)
	}
}
