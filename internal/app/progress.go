package app

import (
	"fmt"
	"strings"
	"sync"
)

const progressBarWidth = 30

type ProgressBar struct {
	mu    sync.Mutex
	total int
	done  int
	label string
}

func newProgressBar(total int, label string) *ProgressBar {
	return &ProgressBar{total: total, label: label}
}

func (p *ProgressBar) Tick(current string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done++
	p.render(current)
}

func (p *ProgressBar) Set(done int, current string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done = done
	p.render(current)
}

func (p *ProgressBar) render(current string) {
	pct := 0
	filled := 0
	if p.total > 0 {
		pct = p.done * 100 / p.total
		filled = progressBarWidth * p.done / p.total
	}
	if filled > progressBarWidth {
		filled = progressBarWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)

	const maxCurrent = 38
	if len(current) > maxCurrent {
		current = "…" + current[len(current)-(maxCurrent-1):]
	}
	fmt.Printf("\r  %-10s [%s] %d/%d (%3d%%)  %s",
		p.label, bar, p.done, p.total, pct, current)
}

func (p *ProgressBar) Finish(summary string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Printf("\r%80s\r", "")
	if summary != "" {
		fmt.Println(summary)
	}
}

type MultiProgressBar struct {
	mu      sync.Mutex
	total   int
	done    int
	label   string
	slots   []string
	started bool
}

func newMultiProgressBar(total int, label string, numSlots int) *MultiProgressBar {
	return &MultiProgressBar{
		total: total,
		label: label,
		slots: make([]string, numSlots),
	}
}

func (p *MultiProgressBar) SetSlot(workerID int, file string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if workerID >= 0 && workerID < len(p.slots) {
		p.slots[workerID] = file
	}
	p.redraw()
}

func (p *MultiProgressBar) Complete(workerID int, file string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done++
	if workerID >= 0 && workerID < len(p.slots) && file != "" {
		p.slots[workerID] = file
	}
	p.redraw()
}

func (p *MultiProgressBar) numLines() int { return 1 + len(p.slots) }

func (p *MultiProgressBar) redraw() {
	n := p.numLines()
	if p.started {
		fmt.Printf("\033[%dA", n)
	}

	pct, filled := 0, 0
	if p.total > 0 {
		pct = p.done * 100 / p.total
		filled = progressBarWidth * p.done / p.total
	}
	if filled > progressBarWidth {
		filled = progressBarWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)
	fmt.Printf("\r\033[2K  %-10s [%s] %d/%d (%3d%%)\n",
		p.label, bar, p.done, p.total, pct)

	for i, slot := range p.slots {
		prefix := "  ├─"
		if i == len(p.slots)-1 {
			prefix = "  └─"
		}
		display := slot
		if display == "" {
			display = "·"
		}
		const maxLen = 52
		if len(display) > maxLen {
			display = "…" + display[len(display)-(maxLen-1):]
		}
		fmt.Printf("\r\033[2K%s [%d] %s\n", prefix, i+1, display)
	}
	p.started = true
}

func (p *MultiProgressBar) Finish(summary string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		n := p.numLines()

		fmt.Printf("\033[%dA", n)
		for i := 0; i < n; i++ {
			fmt.Printf("\r\033[2K\n")
		}
		fmt.Printf("\033[%dA", n)
	}
	if summary != "" {
		fmt.Println(summary)
	}
}
