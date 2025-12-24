package main

import (
	"strings"
	"time"

	"github.com/andlabs/ui"

	"github.com/nulldozer/printer-calibration-utility/printer"
)

type bedLevelTab struct {
	client        *printer.Client
	hint          *ui.Label
	status        *ui.Label
	runBtn        *ui.Button
	validateBtn   *ui.Button
	routineActive bool
	waitCh        chan struct{}
	progressSeen  bool
	savedSeen     bool
	loadedSeen    bool
}

func newBedLevelTab(client *printer.Client) *bedLevelTab {
	t := &bedLevelTab{client: client}
	client.AddBedLevelListener(t.onBedLine)
	return t
}

func (t *bedLevelTab) Build() ui.Control {
	vbox := ui.NewVerticalBox()
	vbox.SetPadded(true)

	t.hint = ui.NewLabel("")
	vbox.Append(t.hint, false)

	group := ui.NewGroup("Bed Leveling")
	group.SetMargined(true)
	groupBox := ui.NewVerticalBox()
	groupBox.SetPadded(true)

	t.runBtn = ui.NewButton("Start Bed Leveling Routine")
	t.runBtn.OnClicked(func(*ui.Button) {
		if t.routineActive {
			return
		}
		t.startRoutine()
	})
	groupBox.Append(t.runBtn, false)

	t.status = ui.NewLabel("")
	groupBox.Append(t.status, false)

	t.validateBtn = ui.NewButton("Print Validation Pattern")
	t.validateBtn.OnClicked(func(*ui.Button) {
		_ = t.client.PrintValidationPattern()
	})
	groupBox.Append(t.validateBtn, false)

	group.SetChild(groupBox)
	vbox.Append(group, false)

	t.OnConnectionChanged(false)
	return vbox
}

func (t *bedLevelTab) setStatus(text string) {
	ui.QueueMain(func() {
		if t.status != nil {
			t.status.SetText(text)
		}
	})
}

func (t *bedLevelTab) startRoutine() {
	t.routineActive = true
	t.progressSeen = false
	t.savedSeen = false
	t.loadedSeen = false
	t.waitCh = make(chan struct{}, 1)
	// Clear any leftover buffered lines so we only react to fresh output.
	t.client.ClearLineBuffer()
	t.setStatus("Running bed leveling routine...")
	if t.runBtn != nil {
		t.runBtn.Disable()
	}
	go func() {
		if err := t.client.RunBedLevelingRoutine(); err != nil {
			t.finishRoutine("Bed leveling failed: " + err.Error())
			return
		}
		select {
		case <-t.waitCh:
			t.finishRoutine("Mesh saved and bed leveling activated.")
		case <-time.After(5 * time.Minute):
			t.finishRoutine("Bed leveling timed out.")
		}
	}()
}

func (t *bedLevelTab) finishRoutine(msg string) {
	if !t.routineActive {
		return
	}
	t.routineActive = false
	ui.QueueMain(func() {
		if t.runBtn != nil {
			t.runBtn.Enable()
		}
		t.setStatus(msg)
	})
}

func (t *bedLevelTab) onBedLine(line string) {
	if !t.routineActive {
		return
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	// skip temp chatter
	if strings.HasPrefix(line, "T:") || strings.HasPrefix(line, "B:") {
		return
	}

	if strings.Contains(line, "Probing mesh point") ||
		strings.Contains(line, "Mesh invalidated") ||
		strings.Contains(line, "Mesh saved") ||
		strings.Contains(line, "Mesh loaded") ||
		strings.Contains(strings.ToLower(line), "leveling") {
		t.setStatus(line)
	}

	if strings.Contains(line, "Mesh invalidated") || strings.Contains(line, "Probing mesh point") {
		t.progressSeen = true
	}

	if t.progressSeen && strings.Contains(line, "Mesh saved") {
		t.savedSeen = true
	}
	if t.progressSeen && strings.Contains(line, "Mesh loaded from slot") {
		t.loadedSeen = true
	}
	if t.savedSeen && (t.loadedSeen || strings.Contains(line, "Done.")) {
		select {
		case t.waitCh <- struct{}{}:
		default:
		}
	}
}

func (t *bedLevelTab) OnConnectionChanged(connected bool) {
	ui.QueueMain(func() {
		if t.hint != nil {
			if connected {
				t.hint.SetText("")
			} else {
				t.hint.SetText("Connect first to run bed leveling.")
			}
		}
		for _, btn := range []*ui.Button{t.runBtn, t.validateBtn} {
			if btn == nil {
				continue
			}
			if connected {
				btn.Enable()
			} else {
				btn.Disable()
			}
		}
		if !connected && t.status != nil {
			t.status.SetText("")
		}
	})
}
