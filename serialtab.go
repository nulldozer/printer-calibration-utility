package main

import (
	"fmt"
	"strings"

	"github.com/andlabs/ui"

	"github.com/nulldozer/printer-calibration-utility/printer"
)

type serialTab struct {
	client     *printer.Client
	hint       *ui.Label
	log        *ui.MultilineEntry
	inputEntry *ui.Entry
	sendBtn    *ui.Button
}

func newSerialTab(client *printer.Client) *serialTab {
	st := &serialTab{client: client}
	client.AddLogListener(st.onLog)
	return st
}

func (t *serialTab) Build() ui.Control {
	vbox := ui.NewVerticalBox()
	vbox.SetPadded(true)

	t.hint = ui.NewLabel("")
	vbox.Append(t.hint, false)

	vbox.Append(t.buildOutputGroup(), true)
	vbox.Append(t.buildInputRow(), false)

	return vbox
}

func (t *serialTab) buildOutputGroup() ui.Control {
	group := ui.NewGroup("Monitor")
	group.SetMargined(true)

	t.log = ui.NewNonWrappingMultilineEntry()
	t.log.SetReadOnly(true)
	group.SetChild(t.log)

	return group
}

func (t *serialTab) buildInputRow() ui.Control {
	box := ui.NewHorizontalBox()
	box.SetPadded(true)

	t.inputEntry = ui.NewEntry()
	t.inputEntry.SetReadOnly(false)
	box.Append(t.inputEntry, true)

	t.sendBtn = ui.NewButton("Send")
	t.sendBtn.Disable()
	t.sendBtn.OnClicked(func(*ui.Button) {
		t.sendLine()
	})
	t.inputEntry.OnChanged(func(e *ui.Entry) {
		text := e.Text()
		if strings.Contains(text, "\n") || strings.Contains(text, "\r") {
			trimmed := strings.TrimSpace(text)
			// remove the newline so the user doesn't see it flash
			e.SetText(trimmed)
			t.sendLine()
		}
	})
	box.Append(t.sendBtn, false)

	return box
}

func (t *serialTab) sendLine() {
	text := strings.TrimSpace(t.inputEntry.Text())
	if text == "" {
		return
	}

	if err := t.client.SendRaw(text); err != nil {
		t.onLog(fmt.Sprintf("Write failed: %v\n", err))
		return
	}
	t.onLog("> " + text + "\n")
	ui.QueueMain(func() {
		t.inputEntry.SetText("")
	})
}

func (t *serialTab) onLog(text string) {
	ui.QueueMain(func() {
		if t.log != nil {
			t.log.Append(text)
		}
	})
}

func (t *serialTab) OnConnectionChanged(connected bool) {
	ui.QueueMain(func() {
		if connected {
			t.sendBtn.Enable()
			t.inputEntry.Enable()
			if t.hint != nil {
				t.hint.SetText("")
			}
		} else {
			t.sendBtn.Disable()
			t.inputEntry.Disable()
			if t.hint != nil {
				t.hint.SetText("Connect first to send commands.")
			}
		}
	})
}
