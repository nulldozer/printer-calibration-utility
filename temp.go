package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/andlabs/ui"

	"l2ui/printer"
)

type tempTab struct {
	client     *printer.Client
	hint       *ui.Label
	hotLabel   *ui.Label
	bedLabel   *ui.Label
	hotEntry   *ui.Entry
	bedEntry   *ui.Entry
	startBtn   *ui.Button
	stopBtn    *ui.Button
	hotBtn     *ui.Button
	bedBtn     *ui.Button
	monitoring bool
}

func newTempTab(client *printer.Client) *tempTab {
	t := &tempTab{client: client}
	client.AddTempListener(t.onTempUpdate)
	return t
}

func (t *tempTab) Build() ui.Control {
	vbox := ui.NewVerticalBox()
	vbox.SetPadded(true)

	t.hint = ui.NewLabel("")
	vbox.Append(t.hint, false)

	btnRow := ui.NewHorizontalBox()
	btnRow.SetPadded(true)
	t.startBtn = ui.NewButton("Start Monitoring")
	t.startBtn.OnClicked(func(*ui.Button) {
		t.monitoring = true
		_ = t.client.StartTempMonitoring()
	})
	t.stopBtn = ui.NewButton("Stop Monitoring")
	t.stopBtn.OnClicked(func(*ui.Button) {
		t.monitoring = false
		_ = t.client.StopTempMonitoring()
		t.resetLabels()
	})
	btnRow.Append(t.startBtn, false)
	btnRow.Append(t.stopBtn, false)
	vbox.Append(btnRow, false)

	grid := ui.NewGrid()
	grid.SetPadded(true)

	t.hotEntry = ui.NewEntry()
	t.hotEntry.SetText("210")
	t.hotBtn = ui.NewButton("Preheat Hotend")
	t.hotBtn.OnClicked(func(*ui.Button) {
		t.preheatHotend()
	})
	t.hotLabel = ui.NewLabel("? / ?")

	t.bedEntry = ui.NewEntry()
	t.bedEntry.SetText("60")
	t.bedBtn = ui.NewButton("Preheat Bed")
	t.bedBtn.OnClicked(func(*ui.Button) {
		t.preheatBed()
	})
	t.bedLabel = ui.NewLabel("? / ?")

	grid.Append(ui.NewLabel("Hotend"), 0, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
	grid.Append(t.hotLabel, 1, 0, 1, 1, false, ui.AlignStart, false, ui.AlignFill)
	grid.Append(t.hotEntry, 2, 0, 1, 1, true, ui.AlignFill, false, ui.AlignFill)
	grid.Append(t.hotBtn, 3, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)

	grid.Append(ui.NewLabel("Bed"), 0, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
	grid.Append(t.bedLabel, 1, 1, 1, 1, false, ui.AlignStart, false, ui.AlignFill)
	grid.Append(t.bedEntry, 2, 1, 1, 1, true, ui.AlignFill, false, ui.AlignFill)
	grid.Append(t.bedBtn, 3, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)

	vbox.Append(grid, false)

	t.OnConnectionChanged(false)
	return vbox
}

func (t *tempTab) onTempUpdate(hCurrent, hTarget, bCurrent, bTarget string) {
	if !t.monitoring {
		return
	}
	ui.QueueMain(func() {
		if t.hotLabel != nil {
			t.hotLabel.SetText(fmt.Sprintf("%s / %s", hCurrent, hTarget))
		}
		if t.bedLabel != nil {
			t.bedLabel.SetText(fmt.Sprintf("%s / %s", bCurrent, bTarget))
		}
	})
}

func (t *tempTab) resetLabels() {
	ui.QueueMain(func() {
		if t.hotLabel != nil {
			t.hotLabel.SetText("? / ?")
		}
		if t.bedLabel != nil {
			t.bedLabel.SetText("? / ?")
		}
	})
}

func (t *tempTab) preheatHotend() {
	val := strings.TrimSpace(t.hotEntry.Text())
	if val == "" {
		return
	}
	temp, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return
	}
	_ = t.client.PreheatHotend(temp)
}

func (t *tempTab) preheatBed() {
	val := strings.TrimSpace(t.bedEntry.Text())
	if val == "" {
		return
	}
	temp, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return
	}
	_ = t.client.PreheatBed(temp)
}

func (t *tempTab) OnConnectionChanged(connected bool) {
	ui.QueueMain(func() {
		if t.hint != nil {
			if connected {
				t.hint.SetText("")
			} else {
				t.hint.SetText("Connect first to control temperatures.")
			}
		}
		for _, btn := range []*ui.Button{t.startBtn, t.stopBtn, t.hotBtn, t.bedBtn} {
			if btn == nil {
				continue
			}
			if connected {
				btn.Enable()
			} else {
				btn.Disable()
			}
		}
	})
}
