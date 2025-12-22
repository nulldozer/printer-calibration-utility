package main

import (
	"fmt"

	"github.com/andlabs/ui"

	"l2ui/printer"
)

type zOffsetTab struct {
	client     *printer.Client
	hint       *ui.Label
	resetBtn   *ui.Button
	applyBtn   *ui.Button
	buttons    []*ui.Button
	currentZ   float64
	stageReady bool
}

func newZOffsetTab(client *printer.Client) *zOffsetTab {
	return &zOffsetTab{client: client}
}

func (t *zOffsetTab) Build() ui.Control {
	t.currentZ = 0
	t.stageReady = false

	vbox := ui.NewVerticalBox()
	vbox.SetPadded(true)
	t.hint = ui.NewLabel("")
	vbox.Append(t.hint, false)

	// Stage 1
	stage1 := ui.NewGroup("Stage 1")
	stage1.SetMargined(true)
	stage1Box := ui.NewVerticalBox()
	stage1Box.SetPadded(true)
	t.resetBtn = ui.NewButton("Reset Z Offset")
	t.resetBtn.OnClicked(func(*ui.Button) {
		go t.runStage1()
	})
	t.resetBtn.Disable()
	stage1Box.Append(t.resetBtn, false)
	stage1Box.Append(ui.NewLabel("Runs: M851 Z0, G28, G0 Z0"), false)
	stage1.SetChild(stage1Box)
	vbox.Append(stage1, false)

	// Stage 2
	stage2 := ui.NewGroup("Stage 2")
	stage2.SetMargined(true)
	stage2Box := ui.NewVerticalBox()
	stage2Box.SetPadded(true)
	stage2Box.Append(ui.NewLabel("Move Z until the nozzle touches the bed"), false)

	t.buttons = nil
	deltas := []float64{1.0, 0.1, 0.05, -0.05, -0.1, -1.0}
	for _, d := range deltas {
		val := d
		btn := ui.NewButton(fmt.Sprintf("%+.2f", val))
		btn.OnClicked(func(*ui.Button) {
			t.adjustZ(val)
		})
		stage2Box.Append(btn, false)
		t.buttons = append(t.buttons, btn)
	}

	stage2.SetChild(stage2Box)
	vbox.Append(stage2, false)

	// Stage 3
	stage3 := ui.NewGroup("Stage 3")
	stage3.SetMargined(true)
	stage3Box := ui.NewVerticalBox()
	stage3Box.SetPadded(true)
	stage3Box.Append(ui.NewLabel("If the nozzle is touching the bed, click Apply"), false)
	t.applyBtn = ui.NewButton("Apply")
	t.applyBtn.OnClicked(func(*ui.Button) {
		go t.applyOffset()
	})
	stage3Box.Append(t.applyBtn, false)
	stage3.SetChild(stage3Box)
	vbox.Append(stage3, false)

	t.enableStage2(false)
	t.OnConnectionChanged(false)

	return vbox
}

func (t *zOffsetTab) runStage1() {
	t.enableStage2(false)
	t.currentZ = 0
	if err := t.client.ResetZOffset(); err != nil {
		return
	}
	t.enableStage2(true)
}

func (t *zOffsetTab) adjustZ(delta float64) {
	if !t.stageReady {
		return
	}
	t.currentZ += delta
	_ = t.client.MoveToZ(t.currentZ)
}

func (t *zOffsetTab) applyOffset() {
	if !t.stageReady {
		return
	}
	if err := t.client.ApplyZOffset(t.currentZ); err != nil {
		return
	}
	_ = t.client.SaveSettings()
	t.enableStage2(false)
}

func (t *zOffsetTab) enableStage2(enable bool) {
	t.stageReady = enable
	ui.QueueMain(func() {
		for _, b := range t.buttons {
			if enable {
				b.Enable()
			} else {
				b.Disable()
			}
		}
		if t.applyBtn != nil {
			if enable {
				t.applyBtn.Enable()
			} else {
				t.applyBtn.Disable()
			}
		}
	})
}

func (t *zOffsetTab) OnConnectionChanged(connected bool) {
	ui.QueueMain(func() {
		if t.resetBtn != nil {
			if connected {
				t.resetBtn.Enable()
			} else {
				t.resetBtn.Disable()
			}
		}
		if t.hint != nil {
			if connected {
				t.hint.SetText("")
			} else {
				t.hint.SetText("Connect first to run Z offset steps.")
			}
		}
	})
}
