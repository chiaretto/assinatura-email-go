package main

import (
	"fmt"
	"image"
	"image/draw"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

// Assets e constantes de composição. As coordenadas são absolutas sobre
// modelo-assinatura.png (1604x287), com origem no canto superior esquerdo.
const (
	modeloPath    = "modelo-assinatura.png"
	arialPath     = "./Arial.ttf"
	arialBoldPath = "./ArialBold.ttf"

	corTexto = "#4C4C4C"

	// Largura mínima do preview. A altura é derivada da proporção real do
	// modelo, para não virar um número mágico que desalinha se o modelo mudar.
	previewLargura = 700

	janelaLargura = 1000
	janelaAltura  = 480
)

// signatureData agrupa os textos que vão sobre o modelo. Existe para evitar
// carregar cinco parâmetros posicionais de string por toda a cadeia de
// renderização.
type signatureData struct {
	name         string
	email        string
	profession   string
	registration string
	telefones    string
}

// renderer guarda em memória tudo que vem do disco. Reparsear ~1 MB de
// TrueType e redecodificar o PNG do modelo a cada assinatura era irrelevante
// quando isso acontecia uma vez por clique, mas domina o custo de qualquer
// renderização repetida — daí o cache.
//
// Não é seguro para uso concorrente: a gg documenta que as font.Face que
// devolve não podem ser usadas em paralelo entre goroutines, e aqui elas são
// compartilhadas por todas as renderizações. Enquanto render() só for chamado
// pela thread de UI, tudo bem — mas renderizar em background (um preview com
// debounce, por exemplo) exige serializar o acesso.
type renderer struct {
	bg *image.RGBA

	faceNome      font.Face // Arial 40
	faceCorpo     font.Face // Arial 30
	faceTelefones font.Face // Arial Bold 25
}

// newRenderer carrega os assets uma única vez. Depois disto, render() não toca
// mais no sistema de arquivos.
func newRenderer() (*renderer, error) {
	bgImage, err := gg.LoadPNG(modeloPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar a imagem de fundo: %v", err)
	}

	// Converter para RGBA já aqui: é o formato que o gg usa internamente, então
	// cada render passa a ser uma cópia direta de memória em vez de uma
	// conversão pixel a pixel do formato original do PNG.
	bounds := bgImage.Bounds()
	bg := image.NewRGBA(bounds)
	draw.Draw(bg, bounds, bgImage, bounds.Min, draw.Src)

	faceNome, err := gg.LoadFontFace(arialPath, 40)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar a fonte: %v", err)
	}

	faceCorpo, err := gg.LoadFontFace(arialPath, 30)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar a fonte: %v", err)
	}

	faceTelefones, err := gg.LoadFontFace(arialBoldPath, 25)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar a fonte: %v", err)
	}

	return &renderer{
		bg:            bg,
		faceNome:      faceNome,
		faceCorpo:     faceCorpo,
		faceTelefones: faceTelefones,
	}, nil
}

// render compõe a assinatura em memória e devolve a imagem resultante. Não
// retorna erro porque todo o I/O já aconteceu em newRenderer.
func (r *renderer) render(d signatureData) image.Image {
	// Uma cópia nova por chamada: o chamador pode reter a imagem devolvida
	// (o preview, por exemplo) enquanto a próxima renderização acontece.
	dst := image.NewRGBA(r.bg.Bounds())
	copy(dst.Pix, r.bg.Pix)

	dc := gg.NewContextForRGBA(dst)
	dc.SetHexColor(corTexto)

	// Atenção ao trocar SetFontFace por LoadFontFace ou vice-versa: os dois
	// calculam dc.fontHeight de formas diferentes (métricas da fonte contra
	// points*72/96). A diferença só chega ao resultado através do termo
	// "y += ay*h" de DrawStringAnchored, e todas as chamadas abaixo usam
	// ay = 0 — por isso os dois caminhos são equivalentes aqui. Um ay
	// diferente de zero passaria a depender desse detalhe.
	dc.SetFontFace(r.faceNome)
	dc.DrawStringAnchored(d.name, 300, 162, 0, 0)

	dc.SetFontFace(r.faceCorpo)
	dc.DrawStringAnchored(juntarProfissao(d.profession, d.registration), 300, 198, 0, 0)
	dc.DrawStringAnchored(d.email, 300, 265, 0, 0)

	dc.SetFontFace(r.faceTelefones)
	dc.DrawStringAnchored(d.telefones, 900, 148, 0, 0)

	return dst
}

// save renderiza e grava a assinatura em disco.
func (r *renderer) save(d signatureData, fileName string) error {
	return gg.SavePNG(fileName, r.render(d))
}

func main() {
	a := app.New()
	w := a.NewWindow("Gerar Assinatura de Email")

	// Assets carregados uma vez, no startup. Uma falha aqui não é fatal: a
	// janela abre em modo degradado (ver montarPreview).
	rend, rendErr := newRenderer()

	// Campos do formulário
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Nome Completo")

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("email@hfcd.com.br")

	professionEntry := widget.NewEntry()
	professionEntry.SetPlaceHolder("Advogado")

	registrationEntry := widget.NewEntry()
	registrationEntry.SetPlaceHolder("OAB/MG - 123456")

	celularEntry := widget.NewEntry()
	celularEntry.SetPlaceHolder("(34) 9 9126-5991")

	telefoneEntry := widget.NewEntry()
	telefoneEntry.SetPlaceHolder("(34) 3232-6699")
	telefoneEntry.SetText("(34) 3232-6699")

	campos := []*widget.Entry{
		nameEntry,
		emailEntry,
		professionEntry,
		registrationEntry,
		celularEntry,
		telefoneEntry,
	}

	// coletarDados é a única leitura dos campos: o preview e a gravação
	// enxergam exatamente o mesmo estado.
	coletarDados := func() signatureData {
		return signatureData{
			name:         nameEntry.Text,
			email:        emailEntry.Text,
			profession:   professionEntry.Text,
			registration: registrationEntry.Text,
			telefones:    juntarTelefones(celularEntry.Text, telefoneEntry.Text),
		}
	}

	// Botão para gerar a assinatura
	generateButton := widget.NewButton("Gerar Assinatura de Email", func() {
		data := coletarDados()

		if data.name == "" || data.email == "" || data.profession == "" {
			dialog.ShowError(fmt.Errorf("Todos os campos devem ser preenchidos"), w)
			return
		}

		if rendErr != nil {
			dialog.ShowError(fmt.Errorf("Erro ao gerar a assinatura: %v", rendErr), w)
			return
		}

		fileName := fmt.Sprintf("assinatura-%s.png", data.name)
		if err := rend.save(data, fileName); err != nil {
			dialog.ShowError(fmt.Errorf("Erro ao gerar a assinatura: %v", err), w)
			return
		}

		// O arquivo já está salvo; se o caminho absoluto não puder ser
		// resolvido, mostrar o relativo é melhor que falhar.
		caminho, err := filepath.Abs(fileName)
		if err != nil {
			caminho = fileName
		}

		dialog.ShowInformation("Sucesso", fmt.Sprintf("Assinatura gerada com sucesso:\n%s", caminho), w)
	})

	// Layout do formulário
	form := container.NewVBox(
		nameEntry,
		emailEntry,
		professionEntry,
		registrationEntry,
		celularEntry,
		telefoneEntry,
		generateButton,
	)

	preview := montarPreview(rend, rendErr, campos, coletarDados, generateButton)

	w.SetContent(container.NewBorder(form, nil, nil, nil, preview))
	w.Resize(fyne.NewSize(janelaLargura, janelaAltura))
	w.ShowAndRun()
}

// montarPreview devolve a área de preview já ligada aos campos do formulário.
// Se os assets não carregaram, devolve no lugar dela uma mensagem de erro e
// desabilita a geração — a janela continua utilizável.
func montarPreview(
	rend *renderer,
	rendErr error,
	campos []*widget.Entry,
	coletarDados func() signatureData,
	generateButton *widget.Button,
) fyne.CanvasObject {
	if rendErr != nil {
		generateButton.Disable()

		aviso := widget.NewLabel(fmt.Sprintf(
			"Não foi possível carregar os arquivos da assinatura:\n\n%v\n\n"+
				"Verifique se %s, %s e %s estão na mesma pasta do aplicativo.",
			rendErr, modeloPath, arialPath, arialBoldPath))
		aviso.Wrapping = fyne.TextWrapWord
		aviso.Alignment = fyne.TextAlignCenter

		return container.NewCenter(aviso)
	}

	// Preview inicial: o app já abre mostrando o modelo, sem exigir interação.
	imagem := canvas.NewImageFromImage(rend.render(coletarDados()))
	imagem.FillMode = canvas.ImageFillContain

	// Altura derivada da proporção real do modelo em vez de um valor fixo.
	bounds := rend.bg.Bounds()
	altura := previewLargura * float32(bounds.Dy()) / float32(bounds.Dx())
	imagem.SetMinSize(fyne.NewSize(previewLargura, altura))

	// Renderizar leva ~3 ms, bem abaixo do intervalo entre teclas, então a
	// atualização é síncrona. Isso é proposital: o Fyne 2.5.1 não tem fyne.Do,
	// e tocar no canvas a partir de outra goroutine não seria seguro.
	atualizar := func(string) {
		imagem.Image = rend.render(coletarDados())
		imagem.Refresh()
	}
	for _, campo := range campos {
		campo.OnChanged = atualizar
	}

	return imagem
}

// juntarTelefones monta a linha de contato, evitando o separador quando um dos
// telefones não foi informado.
func juntarTelefones(celular, telefone string) string {
	return juntarCampos(celular, telefone, " / ")
}

// juntarProfissao monta a linha "profissão - registro". O registro é opcional,
// então sem ele a linha não pode terminar com um traço órfão.
func juntarProfissao(profession, registration string) string {
	return juntarCampos(profession, registration, " - ")
}

// juntarCampos concatena dois textos com o separador, omitindo-o quando um dos
// lados está vazio — inclusive quando vem só com espaços.
func juntarCampos(esquerda, direita, separador string) string {
	esquerda = strings.TrimSpace(esquerda)
	direita = strings.TrimSpace(direita)

	switch {
	case esquerda == "":
		return direita
	case direita == "":
		return esquerda
	default:
		return esquerda + separador + direita
	}
}
