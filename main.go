package main

import (
	"errors"
	"fmt"
	"image/color"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	ignore "github.com/sabhiram/go-gitignore"
	sqdialog "github.com/sqweek/dialog"
)

// =================================================================================
// 1. CONFIGURAÇÃO E MODELOS (Domain Layer)
// =================================================================================

type LanguageWeights struct {
	FileWeight       float64
	OccurrenceWeight float64
}

// Configurações estáticas
var (
	languageExtensions = map[string][]string{
		"Java":       {".java"},
		"Kotlin":     {".kt", ".kts"},
		"Python":     {".py"},
		"SQL":        {".sql"},
		"CSharp":     {".cs"},
		"TypeScript": {".ts", ".tsx"},
		"JavaScript": {".js", ".jsx"},
		"Cobol":      {".cob", ".cbl", ".cpy"},
	}

	languageWeights = map[string]LanguageWeights{
		"Java":       {1.5, 0.2},
		"Kotlin":     {1.5, 0.2},
		"Python":     {1.5, 0.2},
		"SQL":        {3.0, 0.25},
		"CSharp":     {1.5, 0.2},
		"TypeScript": {1.2, 0.15},
		"JavaScript": {1.2, 0.15},
		"Cobol":      {4.0, 0.3},
	}

	defaultVariables = []string{
		"CNPJ", "digitodocto", "documento", "numerodocumento",
		"digitodocumento", "nrodocto", "numerodocto", "numdocto",
		"numdocumento", "nrodocumento", "filialdocto", "filialdocumento", "CGC",
	}

	availableLanguages = []string{
		"Java", "Kotlin", "Python", "SQL", "CSharp", "TypeScript", "JavaScript", "Cobol",
	}
)

// Estruturas de Resultado
type RowData struct {
	Language        string
	Files           int
	FilesWithVar    int
	Occurrences     int
	EstimatedHours  float64 // Mantemos como float para facilidade de soma
	EstimatedMinutes int
}

type AnalysisResult struct {
	TotalFiles    int
	HasDocker     bool
	Rows          []RowData
	TotalMinutes  int
}

// =================================================================================
// 2. LÓGICA DE NEGÓCIO (Service Layer)
// =================================================================================

// LogicService agrupa funções de análise
type LogicService struct{}

func (s *LogicService) LoadGitignore(basePath string) *ignore.GitIgnore {
	gitignorePath := filepath.Join(basePath, ".gitignore")
	ignorer, err := ignore.CompileIgnoreFile(gitignorePath)
	if err != nil {
		return ignore.CompileIgnoreLines()
	}
	return ignorer
}

func (s *LogicService) Analyze(path string, variables []string, languages []string) AnalysisResult {
	basePath, _ := filepath.Abs(path)
	gitIgnore := s.LoadGitignore(basePath)

	stats := make(map[string]*RowData)
	for _, lang := range languages {
		stats[lang] = &RowData{Language: lang}
	}

	totalFiles := 0
	hasDocker := false
	dockerFiles := map[string]bool{"dockerfile": true, "docker-compose.yml": true, "compose.yml": true}

	filepath.WalkDir(basePath, func(currPath string, d fs.DirEntry, err error) error {
		if err != nil { return nil }

		relPath, _ := filepath.Rel(basePath, currPath)
		if d.IsDir() && d.Name() == ".git" { return filepath.SkipDir }
		if gitIgnore.MatchesPath(relPath) {
			if d.IsDir() { return filepath.SkipDir }
			return nil
		}
		if d.IsDir() { return nil }

		fileLower := strings.ToLower(d.Name())
		totalFiles++

		// Checagem Docker
		if strings.HasPrefix(fileLower, "dockerfile") || dockerFiles[fileLower] {
			hasDocker = true
		}

		// Checagem Linguagens
		for _, lang := range languages {
			extensions := languageExtensions[lang]
			matched := false
			for _, ext := range extensions {
				if strings.HasSuffix(fileLower, ext) {
					matched = true
					break
				}
			}

			if matched {
				stats[lang].Files++
				occurrences := s.countOccurrences(currPath, variables)
				stats[lang].Occurrences += occurrences
				if occurrences > 0 {
					stats[lang].FilesWithVar++
				}
				break // Arquivo pertence a uma linguagem apenas (simplificação)
			}
		}
		return nil
	})

	// Processar resultados finais
	var rows []RowData
	totalMinutes := 0

	// Adicionar tempo base de setup e docker
	setupMinutes := 3 * 60 // 3 Horas base
	if !hasDocker {
		setupMinutes += 5 * 60 // +5 horas se não tiver docker
	}
	totalMinutes += setupMinutes

	for _, lang := range languages {
		data := stats[lang]
		if data.Files > 0 {
			w := languageWeights[lang]
			hours := (float64(data.FilesWithVar) * w.FileWeight) + (float64(data.Occurrences) * w.OccurrenceWeight)
			
			minutes := int(math.Round(hours * 60))
			data.EstimatedHours = hours
			data.EstimatedMinutes = minutes
			
			rows = append(rows, *data)
			totalMinutes += minutes
		}
	}

	return AnalysisResult{
		TotalFiles:   totalFiles,
		HasDocker:    hasDocker,
		Rows:         rows,
		TotalMinutes: totalMinutes,
	}
}

func (s *LogicService) countOccurrences(filePath string, variables []string) int {
	if len(variables) == 0 { return 0 }
	contentBytes, err := os.ReadFile(filePath)
	if err != nil { return 0 }
	
	content := strings.ToLower(string(contentBytes))
	total := 0
	for _, v := range variables {
		total += strings.Count(content, strings.ToLower(v))
	}
	return total
}

// Helpers de formatação
func formatMinutesToHHMM(totalMinutes int) string {
	h := totalMinutes / 60
	m := totalMinutes % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

// =================================================================================
// 3. UI COMPONENTS E TEMAS (Presentation Layer)
// =================================================================================

type OrangeTheme struct{}
func (m OrangeTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary {
		return color.NRGBA{R: 242, G: 100, B: 0, A: 255} // #F26400
	}
	return theme.LightTheme().Color(name, variant)
}
func (m OrangeTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return theme.LightTheme().Icon(name) }
func (m OrangeTheme) Font(style fyne.TextStyle) fyne.Resource   { return theme.LightTheme().Font(style) }
func (m OrangeTheme) Size(name fyne.ThemeSizeName) float32      { return theme.LightTheme().Size(name) }

type ReadOnlyEntry struct {
	widget.Entry
}
func NewReadOnlyEntry() *ReadOnlyEntry {
	entry := &ReadOnlyEntry{}
	entry.ExtendBaseWidget(entry)
	return entry
}
func (e *ReadOnlyEntry) TypedRune(r rune) {}
func (e *ReadOnlyEntry) TypedKey(k *fyne.KeyEvent) {}

// =================================================================================
// 4. APLICAÇÃO PRINCIPAL (Controller)
// =================================================================================

type EstimatorApp struct {
	app     fyne.App
	window  fyne.Window
	logic   *LogicService

	// Inputs
	entryDir  *widget.Entry
	entryLang *ReadOnlyEntry
	entryVar  *widget.Entry
	
	// State
	selectedLangs []string
}

func NewEstimatorApp() *EstimatorApp {
	a := app.New()
	a.Settings().SetTheme(&OrangeTheme{})
	
	return &EstimatorApp{
		app:           a,
		window:        a.NewWindow("StackSpot Estimator"),
		logic:         &LogicService{},
		selectedLangs: availableLanguages, // Seleciona tudo por padrão
	}
}

func (e *EstimatorApp) Run() {
	e.buildUI()
	e.window.Resize(fyne.NewSize(700, 550))
	e.window.CenterOnScreen()
	e.window.ShowAndRun()
}

func (e *EstimatorApp) buildUI() {
	// Header
	headerText := canvas.NewText("StackSpot - Estimativa Horas PS", color.Black)
	headerText.TextSize = 24
	headerText.TextStyle = fyne.TextStyle{Bold: true}
	headerContainer := container.NewCenter(headerText)

	// Inputs
	e.entryDir = widget.NewEntry()
	e.entryDir.SetPlaceHolder("Selecione um diretório...")
	
	btnBrowser := widget.NewButton("BROWSER", e.openBrowserDialog)
	dirContainer := container.NewBorder(nil, nil, nil, btnBrowser, e.entryDir)

	e.entryLang = NewReadOnlyEntry()
	e.entryLang.SetText(strings.Join(e.selectedLangs, ", "))
	btnSelectLang := widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), e.openLanguageSelector)
	langContainer := container.NewBorder(nil, nil, nil, btnSelectLang, e.entryLang)

	e.entryVar = widget.NewEntry()
	e.entryVar.SetText(strings.Join(defaultVariables, ", "))
	e.entryVar.MultiLine = true 
	e.entryVar.Wrapping = fyne.TextWrapWord

	// Botão Principal
	btnEstimar := widget.NewButton("ESTIMAR PROJETO", e.runEstimation)
	btnEstimar.Importance = widget.HighImportance

	// Layout Principal
	form := widget.NewForm(
		widget.NewFormItem("Diretório do Projeto", dirContainer),
		widget.NewFormItem("Linguagens Alvo", langContainer),
		widget.NewFormItem("Variáveis (Busca)", e.entryVar),
	)

	content := container.NewVBox(
		headerContainer,
		layout.NewSpacer(),
		form,
		layout.NewSpacer(),
		container.NewPadded(btnEstimar),
	)

	e.window.SetContent(container.NewPadded(content))
}

// Actions
func (e *EstimatorApp) openBrowserDialog() {
	directory, err := sqdialog.Directory().Title("Selecione o diretório").Browse()
	if err == nil {
		e.entryDir.SetText(directory)
	}
}

func (e *EstimatorApp) openLanguageSelector() {
	checkGroup := widget.NewCheckGroup(availableLanguages, func(s []string) {
		e.selectedLangs = s
	})
	checkGroup.SetSelected(e.selectedLangs)

	scroll := container.NewVScroll(checkGroup)
	scroll.SetMinSize(fyne.NewSize(200, 300))

	dialog.NewCustomConfirm("Selecione as Linguagens", "OK", "Cancelar", scroll, func(confirm bool) {
		if confirm {
			e.entryLang.SetText(strings.Join(e.selectedLangs, ", "))
		}
	}, e.window).Show()
}

func (e *EstimatorApp) runEstimation() {
	path := e.entryDir.Text
	if path == "" {
		dialog.ShowError(errors.New("Selecione um diretório válido"), e.window)
		return
	}

	vars := strings.Split(e.entryVar.Text, ",")
	for i := range vars { vars[i] = strings.TrimSpace(vars[i]) }

	// Processamento
	loading := dialog.NewCustom("Processando...", "Aguarde", widget.NewProgressBarInfinite(), e.window)
	loading.Show()

	// Executa em goroutine para não travar UI, mas aqui é síncrono por simplicidade da demo
	result := e.logic.Analyze(path, vars, e.selectedLangs)
	
	loading.Hide()
	e.showReport(result)
}

func (e *EstimatorApp) showReport(res AnalysisResult) {
	// 1. Info Cards
	lblFiles := canvas.NewText(fmt.Sprintf("📁 Arquivos Totais: %d", res.TotalFiles), color.Black)
	
	var lblDocker *canvas.Text
	if res.HasDocker {
		lblDocker = canvas.NewText("🐳 Docker encontrado (Setup Otimizado)", color.NRGBA{0, 150, 0, 255})
	} else {
		lblDocker = canvas.NewText("⚠️ Sem evidência de Docker (+5h Setup)", color.NRGBA{200, 0, 0, 255})
	}
	lblDocker.TextStyle = fyne.TextStyle{Bold: true}

	// 2. Tabela de Resultados
	grid := container.NewGridWithColumns(5)
	
	// Headers
	addCell := func(t string, bold bool) {
		l := widget.NewLabel(t)
		l.Alignment = fyne.TextAlignCenter
		if bold { l.TextStyle = fyne.TextStyle{Bold: true} }
		grid.Add(l)
	}

	headers := []string{"Linguagem", "Arq.", "Arq. c/ Var", "Ocorrências", "Tempo"}
	for _, h := range headers { addCell(h, true) }

	totalFiles := 0
	totalFilesVar := 0
	totalOccur := 0

	for _, row := range res.Rows {
		totalFiles += row.Files
		totalFilesVar += row.FilesWithVar
		totalOccur += row.Occurrences

		addCell(row.Language, false)
		addCell(fmt.Sprintf("%d", row.Files), false)
		addCell(fmt.Sprintf("%d", row.FilesWithVar), false)
		addCell(fmt.Sprintf("%d", row.Occurrences), false)
		addCell(formatMinutesToHHMM(row.EstimatedMinutes), false)
	}

	// Linha Total da Tabela
	addCell("SOMA PARCIAL", true)
	addCell(fmt.Sprintf("%d", totalFiles), true)
	addCell(fmt.Sprintf("%d", totalFilesVar), true)
	addCell(fmt.Sprintf("%d", totalOccur), true)
	addCell(fmt.Sprintf("%s", formatMinutesToHHMM(res.TotalMinutes)), true)

	// 3. Footer Total
	totalStr := formatMinutesToHHMM(res.TotalMinutes)
	lblTotal := canvas.NewText(fmt.Sprintf("⏱ ESTIMATIVA FINAL: %s horas", totalStr), color.Black)
	lblTotal.TextSize = 20
	lblTotal.TextStyle = fyne.TextStyle{Bold: true}

	// Montagem do Dialog
	content := container.NewVBox(
		lblFiles,
		lblDocker,
		layout.NewSpacer(),
		widget.NewCard("Detalhamento", "", container.NewPadded(grid)),
		layout.NewSpacer(),
		container.NewCenter(lblTotal),
	)

	scroll := container.NewVScroll(container.NewPadded(content))
	scroll.SetMinSize(fyne.NewSize(600, 400))

	d := dialog.NewCustom("Relatório de Estimativa", "Fechar", scroll, e.window)
	d.Resize(fyne.NewSize(650, 500))
	d.Show()
}

// =================================================================================
// 5. MAIN ENTRY POINT
// =================================================================================

func main() {
	myApp := NewEstimatorApp()
	myApp.Run()
}