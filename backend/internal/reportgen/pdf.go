package reportgen

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/go-pdf/fpdf"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/store"
)

// Core-шрифти fpdf не підтримують UTF-8, тому кирилиця транслітерується.
var translitMap = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "h", 'ґ': "g", 'д': "d", 'е': "e", 'є': "ie",
	'ж': "zh", 'з': "z", 'и': "y", 'і': "i", 'ї': "yi", 'й': "i", 'к': "k", 'л': "l",
	'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch", 'ь': "",
	'ю': "iu", 'я': "ia", '\'': "",
}

func translit(s string) string {
	var b strings.Builder
	for _, r := range s {
		repl, ok := translitMap[unicode.ToLower(r)]
		if !ok {
			b.WriteRune(r)
			continue
		}
		if unicode.IsUpper(r) && repl != "" {
			b.WriteString(strings.ToUpper(repl[:1]) + repl[1:])
		} else {
			b.WriteString(repl)
		}
	}
	return b.String()
}

func cut(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func money(cents int64, currency string) string {
	if cents <= 0 {
		return "-"
	}
	if currency == "" {
		return fmt.Sprintf("%d", cents/100)
	}
	return fmt.Sprintf("%d %s", cents/100, currency)
}

var palette = [][3]int{
	{63, 207, 142}, {255, 181, 71}, {91, 140, 255}, {163, 113, 247},
	{46, 168, 122}, {255, 107, 122}, {143, 156, 178}, {210, 153, 34},
}

func setFill(pdf *fpdf.Fpdf, c [3]int) { pdf.SetFillColor(c[0], c[1], c[2]) }

type slice struct {
	Label string
	Value int
	Color [3]int
}

// drawDonut — кільцева діаграма з секторів-полігонів.
func drawDonut(pdf *fpdf.Fpdf, cx, cy, rOut, rIn float64, parts []slice) {
	total := 0
	for _, p := range parts {
		total += p.Value
	}
	if total == 0 {
		return
	}
	start := -math.Pi / 2
	for _, p := range parts {
		if p.Value <= 0 {
			continue
		}
		swir := 2 * math.Pi * float64(p.Value) / float64(total)
		setFill(pdf, p.Color)
		pdf.Polygon(donutPoints(cx, cy, rOut, rIn, start, start+swir), "F")
		start += swir
	}
}

func donutPoints(cx, cy, rOut, rIn, a0, a1 float64) []fpdf.PointType {
	steps := int((a1-a0)/(math.Pi/36)) + 1
	if steps < 2 {
		steps = 2
	}
	pts := make([]fpdf.PointType, 0, steps*2+2)
	for i := 0; i <= steps; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(steps)
		pts = append(pts, fpdf.PointType{X: cx + rOut*math.Cos(a), Y: cy + rOut*math.Sin(a)})
	}
	for i := steps; i >= 0; i-- {
		a := a0 + (a1-a0)*float64(i)/float64(steps)
		pts = append(pts, fpdf.PointType{X: cx + rIn*math.Cos(a), Y: cy + rIn*math.Sin(a)})
	}
	return pts
}

func heading(pdf *fpdf.Fpdf, text string) {
	pdf.Ln(2)
	pdf.SetFont("Arial", "B", 12.5)
	pdf.SetTextColor(25, 30, 40)
	pdf.CellFormat(0, 7, translit(text), "", 1, "L", false, 0, "")
	y := pdf.GetY()
	pdf.SetDrawColor(63, 207, 142)
	pdf.SetLineWidth(0.5)
	pdf.Line(15, y, 195, y)
	pdf.SetLineWidth(0.2)
	pdf.Ln(4)
}

func emptyNote(pdf *fpdf.Fpdf, text string) {
	pdf.SetFont("Arial", "", 9.5)
	pdf.SetTextColor(130, 130, 130)
	pdf.CellFormat(0, 6, translit(text), "", 1, "L", false, 0, "")
}

func drawTitle(pdf *fpdf.Fpdf, r store.PDFRunReport) {
	pdf.SetFont("Arial", "B", 17)
	pdf.SetTextColor(25, 30, 40)
	pdf.CellFormat(0, 10, translit(fmt.Sprintf("Звіт прогона пошуку #%d", r.RunID)), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10.5)
	pdf.SetTextColor(70, 70, 70)
	lines := []string{
		fmt.Sprintf("Профіль: %s", r.ProfileName),
		fmt.Sprintf("Статус прогона: %s", r.Status),
		fmt.Sprintf("Замовлень у звіті: %d", len(r.Jobs)),
		fmt.Sprintf("Згенеровано: %s", time.Now().Format("02.01.2006 15:04")),
	}
	if r.StartedAt != nil {
		lines = append(lines, "Почато: "+r.StartedAt.Format("02.01.2006 15:04"))
	}
	if r.FinishedAt != nil {
		lines = append(lines, "Завершено: "+r.FinishedAt.Format("02.01.2006 15:04"))
	}
	for _, line := range lines {
		pdf.CellFormat(0, 6, translit(line), "", 1, "L", false, 0, "")
	}
	pdf.Ln(2)
}

func drawSummaryCards(pdf *fpdf.Fpdf, r store.PDFRunReport) {
	heading(pdf, "Зведення")
	avgBudget := "-"
	if r.Budget.WithBudget > 0 && r.Budget.Avg > 0 {
		avgBudget = money(r.Budget.Avg, "")
	}
	cards := [][2]string{
		{"Замовлень", fmt.Sprintf("%d", len(r.Jobs))},
		{"Дублікатів", fmt.Sprintf("%d", r.Duplicates)},
		{"Заявок", fmt.Sprintf("%d", r.Applications)},
		{"Сер. бюджет", avgBudget},
	}
	const cardW, cardH, gap = 43.5, 16, 2
	y := pdf.GetY()
	for i, c := range cards {
		x := 15 + float64(i)*(cardW+gap)
		pdf.SetFillColor(240, 244, 250)
		pdf.Rect(x, y, cardW, cardH, "F")
		pdf.SetXY(x+3, y+2.5)
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(110, 118, 132)
		pdf.CellFormat(cardW-6, 4, translit(c[0]), "", 2, "L", false, 0, "")
		pdf.SetFont("Arial", "B", 12.5)
		pdf.SetTextColor(25, 30, 40)
		pdf.CellFormat(cardW-6, 6, translit(c[1]), "", 0, "L", false, 0, "")
	}
	pdf.SetY(y + cardH + 2)
}

func drawSourceChart(pdf *fpdf.Fpdf, r store.PDFRunReport) {
	heading(pdf, "Розподіл за джерелами")
	if len(r.Sources) == 0 {
		emptyNote(pdf, "Джерела не знайдено.")
		return
	}
	parts := make([]slice, 0, len(r.Sources))
	for i, s := range r.Sources {
		parts = append(parts, slice{Label: s.Source, Value: s.Count, Color: palette[i%len(palette)]})
	}
	total := 0
	for _, p := range parts {
		total += p.Value
	}
	cy := pdf.GetY() + 30
	drawDonut(pdf, 15+32, cy, 27, 15, parts)
	ly := cy - float64(len(parts))*4.2
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(60, 66, 78)
	for i, p := range parts {
		share := 0.0
		if total > 0 {
			share = float64(p.Value) / float64(total) * 100
		}
		setFill(pdf, p.Color)
		pdf.Rect(15+72, ly+float64(i)*8.4+0.6, 4, 4, "F")
		pdf.SetXY(15+78, ly+float64(i)*8.4-1.2)
		pdf.CellFormat(100, 5, fmt.Sprintf("%s — %d (%.1f%%)", translit(cut(p.Label, 32)), p.Value, share), "", 0, "L", false, 0, "")
	}
	pdf.SetY(cy + 34)
}

func drawDailyChart(pdf *fpdf.Fpdf, r store.PDFRunReport) {
	heading(pdf, "Динаміка надходжень (останні 30 днів)")
	if len(r.Daily) == 0 {
		emptyNote(pdf, "Немає даних за період.")
		return
	}
	maxCount := 1
	for _, p := range r.Daily {
		if p.Count > maxCount {
			maxCount = p.Count
		}
	}
	const x, w, h = 15, 180, 44
	base := pdf.GetY() + h
	plotH := h - 6
	pdf.SetDrawColor(180, 186, 196)
	pdf.SetLineWidth(0.2)
	pdf.Line(x, base, x+w, base)
	barW := w / float64(len(r.Daily))
	for i, p := range r.Daily {
		bh := float64(p.Count) / float64(maxCount) * plotH
		if bh < 0.4 {
			bh = 0.4
		}
		setFill(pdf, [3]int{91, 140, 255})
		pdf.Rect(x+float64(i)*barW+0.6, base-bh, barW-1.2, bh, "F")
	}
	pdf.SetFont("Arial", "", 6.5)
	pdf.SetTextColor(120, 128, 140)
	step := len(r.Daily)/6 + 1
	for i := 0; i < len(r.Daily); i += step {
		label := r.Daily[i].Day
		if len(label) >= 10 {
			label = label[5:]
		}
		pdf.SetXY(x+float64(i)*barW, base+1)
		pdf.CellFormat(barW, 4, label, "", 0, "L", false, 0, "")
	}
	pdf.SetFont("Arial", "B", 8.5)
	pdf.SetTextColor(60, 66, 78)
	pdf.SetXY(x, base-h-3)
	pdf.CellFormat(w, 4, translit(fmt.Sprintf("Макс за добу: %d", maxCount)), "", 0, "R", false, 0, "")
	pdf.SetY(base + 8)
}

func drawHBars(pdf *fpdf.Fpdf, y float64, rows []slice) {
	maxValue := 1
	for _, row := range rows {
		if row.Value > maxValue {
			maxValue = row.Value
		}
	}
	const barH, gap = 5.5, 2.2
	for _, row := range rows {
		pdf.SetFont("Arial", "", 8.5)
		pdf.SetTextColor(60, 66, 78)
		pdf.SetXY(15, y-1.4)
		pdf.CellFormat(60, 4, translit(cut(row.Label, 38)), "", 0, "L", false, 0, "")
		pdf.SetXY(78, y-1.4)
		pdf.CellFormat(15, 4, fmt.Sprintf("%d", row.Value), "", 0, "L", false, 0, "")
		barW := 95 * float64(row.Value) / float64(maxValue)
		setFill(pdf, row.Color)
		pdf.Rect(93, y, barW, barH-2.2, "F")
		y += barH + gap
	}
}

func drawStatusChart(pdf *fpdf.Fpdf, r store.PDFRunReport) {
	if len(r.Statuses) == 0 {
		return
	}
	heading(pdf, "Розподіл за статусами")
	rows := make([]slice, 0, len(r.Statuses))
	for i, st := range r.Statuses {
		rows = append(rows, slice{Label: st.Label, Value: st.Count, Color: palette[i%len(palette)]})
	}
	drawHBars(pdf, pdf.GetY()+2, rows)
	pdf.Ln(float64(len(rows))*7.7 + 6)
}

func drawBudgetTable(pdf *fpdf.Fpdf, r store.PDFRunReport) {
	heading(pdf, "Бюджетна статистика")
	labels := []string{"Мін", "Медіана", "Середній", "Макс", "З бюджетом"}
	vals := []string{
		money(r.Budget.Min, ""),
		money(r.Budget.Median, ""),
		money(r.Budget.Avg, ""),
		money(r.Budget.Max, ""),
		fmt.Sprintf("%d", r.Budget.WithBudget),
	}
	pdf.SetFont("Arial", "B", 8.5)
	pdf.SetFillColor(232, 236, 242)
	pdf.SetTextColor(30, 30, 30)
	for _, l := range labels {
		pdf.CellFormat(36, 7, translit(l), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Arial", "", 9.5)
	for _, v := range vals {
		pdf.CellFormat(36, 7, v, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
}

func drawSkillsChart(pdf *fpdf.Fpdf, r store.PDFRunReport) {
	if len(r.Skills) == 0 {
		return
	}
	heading(pdf, "Топ навичок у знайдених замовленнях")
	rows := make([]slice, 0, len(r.Skills))
	for i, sk := range r.Skills {
		rows = append(rows, slice{Label: sk.Skill, Value: sk.Count, Color: palette[i%len(palette)]})
	}
	drawHBars(pdf, pdf.GetY()+2, rows)
	pdf.Ln(float64(len(rows))*7.7 + 6)
}

func buildConclusions(r store.PDFRunReport) []string {
	var out []string
	if len(r.Sources) > 0 {
		s := r.Sources[0]
		out = append(out, fmt.Sprintf("Найпродуктивніше джерело — %s: %d замовлень (%.1f%% від усіх знайдених).", s.Source, s.Count, s.Share))
	}
	if len(r.Daily) > 0 {
		best := r.Daily[0]
		for _, p := range r.Daily {
			if p.Count > best.Count {
				best = p
			}
		}
		out = append(out, fmt.Sprintf("Найбільше замовлень надійшло %s — %d за добу.", best.Day, best.Count))
	}
	if r.Budget.WithBudget > 0 {
		out = append(out, fmt.Sprintf("Бюджет вказано у %d замовленнях; медіана — %s, у середньому — %s, максимум — %s.",
			r.Budget.WithBudget, money(r.Budget.Median, ""), money(r.Budget.Avg, ""), money(r.Budget.Max, "")))
	}
	if len(r.Jobs) > 0 {
		dup := float64(r.Duplicates) / float64(len(r.Jobs)) * 100
		apps := float64(r.Applications) / float64(len(r.Jobs)) * 100
		out = append(out, fmt.Sprintf("Дублікати складають %.1f%% знахідок; заявки подано на %.1f%% замовлень.", dup, apps))
	}
	if len(r.Skills) > 0 {
		names := make([]string, 0, 3)
		for i, sk := range r.Skills {
			if i >= 3 {
				break
			}
			names = append(names, sk.Skill)
		}
		out = append(out, fmt.Sprintf("Найпопулярніші навички: %s.", strings.Join(names, ", ")))
	}
	if len(out) == 0 {
		out = append(out, "Недостатньо даних для висновків.")
	}
	return out
}

func drawConclusions(pdf *fpdf.Fpdf, r store.PDFRunReport) {
	heading(pdf, "Висновки")
	pdf.SetFont("Arial", "", 9.5)
	pdf.SetTextColor(50, 56, 66)
	for _, line := range buildConclusions(r) {
		pdf.CellFormat(0, 6.2, "- "+translit(line), "", 1, "L", false, 0, "")
	}
}

func jobHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 8.5)
	pdf.SetFillColor(232, 236, 242)
	pdf.SetTextColor(30, 30, 30)
	pdf.CellFormat(8, 7, "#", "1", 0, "R", true, 0, "")
	pdf.CellFormat(93, 7, translit("Замовлення"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(25, 7, translit("Джерело"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(24, 7, translit("Статус"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 7, translit("Бюджет"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(10, 7, translit("Бал"), "1", 1, "R", true, 0, "")
}

func drawJobsTable(pdf *fpdf.Fpdf, r store.PDFRunReport) {
	pdf.AddPage()
	heading(pdf, "Замовлення прогона (за пріоритетом)")
	jobHeader(pdf)
	pdf.SetFont("Arial", "", 8.5)
	for i, job := range r.Jobs {
		if pdf.GetY() > 268 {
			pdf.AddPage()
			jobHeader(pdf)
			pdf.SetFont("Arial", "", 8.5)
		}
		if i%2 == 1 {
			pdf.SetFillColor(246, 248, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.CellFormat(8, 6.4, fmt.Sprintf("%d", i+1), "1", 0, "R", true, 0, "")
		pdf.CellFormat(93, 6.4, translit(cut(job.Title, 56)), "1", 0, "L", true, 0, "")
		pdf.CellFormat(25, 6.4, translit(cut(job.Source, 14)), "1", 0, "L", true, 0, "")
		pdf.CellFormat(24, 6.4, translit(cut(job.StatusLabel, 14)), "1", 0, "L", true, 0, "")
		pdf.CellFormat(20, 6.4, money(job.BudgetCents, job.Currency), "1", 0, "R", true, 0, "")
		pdf.CellFormat(10, 6.4, fmt.Sprintf("%d", job.Priority), "1", 1, "R", true, 0, "")
	}
	if len(r.Jobs) == 0 {
		pdf.CellFormat(0, 8, translit("Замовлень не знайдено."), "1", 1, "L", false, 0, "")
	}
}

// BuildRunReport — детальний аналітичний PDF-звіт прогона пошуку:
// титул, зведення, кругова за джерелами, динаміка по днях, статуси,
// бюджетна статистика, топ-навички, висновки і таблиця замовлень.
func BuildRunReport(report store.PDFRunReport) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(translit(fmt.Sprintf("JuniorMayWork: звіт прогона #%d", report.RunID)), false)
	pdf.SetMargins(15, 16, 15)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-14)
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(130, 130, 130)
		pdf.CellFormat(0, 8, fmt.Sprintf("JuniorMayWork — storinka %d · %s", pdf.PageNo(), time.Now().Format("02.01.2006 15:04")), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()
	drawTitle(pdf, report)
	drawSummaryCards(pdf, report)
	drawSourceChart(pdf, report)
	drawDailyChart(pdf, report)
	drawStatusChart(pdf, report)
	drawBudgetTable(pdf, report)
	drawSkillsChart(pdf, report)
	drawConclusions(pdf, report)
	drawJobsTable(pdf, report)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
