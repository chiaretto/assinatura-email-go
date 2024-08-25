package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/fogleman/gg"
)

func main() {
	a := app.New()
	w := a.NewWindow("Gerar Assinatura de Email")

	// Campos do formulário
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Nome Completo")

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("Email: email@hfcd.com.br")

	professionEntry := widget.NewEntry()
	professionEntry.SetPlaceHolder("Profissão: Advogado")

	registrationEntry := widget.NewEntry()
	registrationEntry.SetPlaceHolder("Número de Registro: OAB/MG - 123456")

	// Botão para gerar a assinatura
	generateButton := widget.NewButton("Gerar Assinatura de Email", func() {
		name := nameEntry.Text
		email := emailEntry.Text
		profession := professionEntry.Text
		registration := registrationEntry.Text

		if name == "" || email == "" || profession == "" {
			dialog.ShowError(fmt.Errorf("Todos os campos devem ser preenchidos"), w)
			return
		}

		fileName := fmt.Sprintf("assinatura-%s.png", name)
		err := generateSignatureImage(name, email, profession, registration, fileName)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Erro ao gerar a assinatura: %v", err), w)
			return
		}

		dialog.ShowInformation("Sucesso", fmt.Sprintf("Assinatura gerada com sucesso: %s", fileName), w)
	})

	// Layout do formulário
	form := container.NewVBox(
		nameEntry,
		emailEntry,
		professionEntry,
		registrationEntry,
		generateButton,
	)

	w.SetContent(form)
	w.Resize(fyne.NewSize(400, 300))
	w.ShowAndRun()
}

func generateSignatureImage(name, email, profession, registration, fileName string) error {
	const width = 2465
	const height = 439

	// Carregar a imagem de fundo
	bgImage, err := gg.LoadPNG("modelo-assinatura.png")
	if err != nil {
		return fmt.Errorf("erro ao carregar a imagem de fundo: %v", err)
	}

	dc := gg.NewContextForImage(bgImage)

	// Definir a cor do texto
	dc.SetHexColor("#4C4C4C")

	// Desenhar o texto na imagem
	err = dc.LoadFontFace("./Arial.ttf", 40)
	if err != nil {
		return fmt.Errorf("erro ao carregar a fonte: %v", err)
	}
	dc.DrawStringAnchored(name, 300, 162, 0, 0)

	err = dc.LoadFontFace("./Arial.ttf", 30)
	if err != nil {
		return fmt.Errorf("erro ao carregar a fonte: %v", err)
	}
	dc.DrawStringAnchored(profession+" - "+registration, 300, 198, 0, 0)

	dc.DrawStringAnchored(email, 300, 265, 0, 0)

	// Salvar a imagem resultante
	return dc.SavePNG(fileName)
}
