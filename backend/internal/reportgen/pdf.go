package reportgen

import (
	"bytes"
	"fmt"
	"strings"
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

func BuildRunReport(report store.PDFRunReport) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(translit(fmt.Sprintf("JuniorMayWork: звіт прогона #%d", report.RunID)), false)
	pdf.SetMargins(15, 16, 15)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-14)
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(130, 130, 130)
		pdf.CellFormat(0, 8, fmt.Sprintf("JuniorMayWork - storinka %d", pdf.PageNo()), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()
	pdf.SetFont("Arial", "B", 17)
	pdf.SetTextColor(25, 30, 40)
	pdf.CellFormat(0, 10, translit(fmt.Sprintf("Звіт прогона пошуку #%d", report.RunID)), "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(70, 70, 70)
	lines := []string{
		translit(fmt.Sprintf("Профіль: %s", report.ProfileName)),
		translit(fmt.Sprintf("Статус: %s", report.Status)),
		translit(fmt.Sprintf("Замовлень у звіті: %d", len(report.Jobs))),
	}
	if report.StartedAt != nil {
		lines = append(lines, translit("Почато: "+report.StartedAt.Format("02.01.2006 15:04")))
	}
	if report.FinishedAt != nil {
		lines = append(lines, translit("Завершено: "+report.FinishedAt.Format("02.01.2006 15:04")))
	}
	for _, line := range lines {
		pdf.CellFormat(0, 7, line, "", 1, "L", false, 0, "")
	}
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(30, 30, 30)
	pdf.SetFillColor(232, 236, 242)
	pdf.CellFormat(95, 8, translit("Замовлення"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(30, 8, translit("Джерело"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(30, 8, translit("Бюджет"), "1", 0, "R", true, 0, "")
	pdf.CellFormat(20, 8, translit("Бал"), "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 9)
	for i, job := range report.Jobs {
		if i%2 == 1 {
			pdf.SetFillColor(246, 248, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.CellFormat(95, 7, translit(cut(job.Title, 58)), "1", 0, "L", true, 0, "")
		pdf.CellFormat(30, 7, translit(cut(job.Source, 16)), "1", 0, "L", true, 0, "")
		budget := "-"
		if job.BudgetCents > 0 {
			budget = fmt.Sprintf("%d %s", job.BudgetCents/100, job.Currency)
		}
		pdf.CellFormat(30, 7, translit(budget), "1", 0, "R", true, 0, "")
		pdf.CellFormat(20, 7, fmt.Sprintf("%d", job.Priority), "1", 1, "R", true, 0, "")
	}
	if len(report.Jobs) == 0 {
		pdf.CellFormat(0, 8, translit("Замовлень не знайдено."), "1", 1, "L", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
