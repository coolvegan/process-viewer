package main

import (
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

type Process struct {
	pText       string
	pId         int32
	Connections []net.ConnectionStat
}

const (
	APP_WIDTH        = 400
	APP_HEIGHT       = 600
	APP_TITLE        = "Process Viewer"
	APP_FIND_PROCESS = "Find your Process..."
	KILL_PROCESS     = "Kill Process"
	POPUP_SHOW       = "SHOW"
)

func processConnections(pList *[]Process) {
	connStates := make(map[int32][]net.ConnectionStat)
	connections, err := net.Connections("all")
	if err != nil {
		fmt.Printf("Fehler beim Abrufen der Verbindungen: %v\n", err)
	}

	for _, p := range *pList {
		for _, conn := range connections {
			if conn.Pid != p.pId || conn.Status == "NONE" {
				continue
			}
			slice := connStates[conn.Pid]
			if slice == nil {
				connStates[conn.Pid] = make([]net.ConnectionStat, 0, 3)
			}
			connStates[conn.Pid] = append(connStates[conn.Pid], conn)
		}
	}

	for pid, cs := range connStates {
		if len(cs) > 0 {
			for i := range *pList {
				if (*pList)[i].pId != pid {
					continue
				}
				(*pList)[i].Connections = append((*pList)[i].Connections, cs...)
			}
		}
	}

}

func processOutput(filter string) *[]Process {
	var processList []Process
	processes, err := process.Processes()
	if err != nil {
		fmt.Println("Fehler beim Abrufen der Prozesse:", err)
		//todo return error
		return nil
	}

	// Informationen zu jedem Prozess anzeigen
	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			name = "Unbekannt"
		}
		pid := p.Pid
		processInfo := fmt.Sprintf("PID: %d, Name: %s", pid, name)
		//Todo Parser einbauen für bessere Filtermöglichkeiten
		if len(filter) > 0 {
			if strings.Index(processInfo, filter) != -1 {
				processList = append(processList, Process{pText: processInfo, pId: pid})
			}
		} else {
			processList = append(processList, Process{pText: processInfo, pId: pid})
		}
	}
	return &processList
}

func main() {
	pFilter := ""
	var content *fyne.Container
	a := app.New()
	a.Settings().SetTheme(&Theme{})
	w := a.NewWindow(APP_TITLE)
	data := []Process{}

	entry := widget.NewEntry()
	entry.SetPlaceHolder(APP_FIND_PROCESS)

	readProcessCallback := func(text string) {
		pFilter = text
		data = *processOutput(pFilter)
		processConnections(&data)

		content.Refresh()
		w.Canvas().Focus(entry)
		w.Show()
	}

	entry.OnChanged = readProcessCallback

	// Liste erstellen
	list := widget.NewList(
		func() int {
			return len(data) // Anzahl der Elemente in der Liste
		},
		func() fyne.CanvasObject {
			//return widget.NewLabel("") // Ein leeres Label als Vorlage
			return container.NewVBox(widget.NewLabel(""))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			vbox := o.(*fyne.Container)
			label := vbox.Objects[0].(*widget.Label)
			////label := o.(*widget.Label)
			label.SetText(data[i].pText) // Daten für jedes Element setzen
			if count := len(data[i].Connections); count > 0 {
				label.SetText(label.Text + fmt.Sprintf(" [Open Ports -> %d]", count))
			}
			label.ExtendBaseWidget(label)

		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		var listposition = &fyne.Position{}
		mi := fyne.NewMenuItem(KILL_PROCESS, nil)
		//Todo find out how to get mouse position
		listposition.X = w.Canvas().Size().Width / 2
		listposition.Y = w.Canvas().Size().Height / 2
		menu := fyne.NewMenu("contextmenu", mi)

		mi.Action = func() {
			process, err := os.FindProcess(int(data[id].pId))
			if err != nil {
				fmt.Println(err)
			}
			err = process.Kill()
			if err != nil {
				fmt.Println(err)
			}
			pFilter = ""
			entry.Text = pFilter
			readProcessCallback(pFilter)
		}

		popup := widget.NewPopUpMenu(menu, w.Canvas())
		popup.ShowAtPosition(*listposition)
	}

	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("ProcessOutput",
			fyne.NewMenuItem(POPUP_SHOW, func() {
				readProcessCallback(pFilter)
			}))
		desk.SetSystemTrayMenu(m)
	}

	content = container.NewBorder(entry, nil, nil, nil, list)
	w.SetContent(content)
	w.SetCloseIntercept(func() {
		w.Hide()
	})
	w.CenterOnScreen()
	w.Resize(fyne.NewSize(APP_WIDTH, APP_HEIGHT))
	w.Canvas().Focus(entry)
	//workaround to start hidden
	//todo make it optional on startup
	//go func() {
	//	time.Sleep(20 * time.Millisecond)
	//	w.Hide()
	//}()

	//Fill ListView
	readProcessCallback(pFilter)
	w.ShowAndRun()
}
