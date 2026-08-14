package main

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/fogleman/gg"
)

// montarPreviewParaTeste monta a área de preview ligada a um único campo de
// nome, que é o suficiente para exercitar o ciclo digitar → renderizar.
func montarPreviewParaTeste(t *testing.T) (*canvas.Image, *widget.Entry, *renderer) {
	t.Helper()
	test.NewApp()

	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}

	nome := widget.NewEntry()
	coletar := func() signatureData {
		return signatureData{name: nome.Text}
	}

	obj := montarPreview(r, nil, []*widget.Entry{nome}, coletar, widget.NewButton("gerar", func() {}))

	imagem, ok := obj.(*canvas.Image)
	if !ok {
		t.Fatalf("esperava *canvas.Image, obtive %T", obj)
	}

	return imagem, nome, r
}

// TestPreviewInicialJaRenderizado cobre o requisito de que o app abre já
// mostrando a assinatura, sem exigir interação.
func TestPreviewInicialJaRenderizado(t *testing.T) {
	imagem, _, r := montarPreviewParaTeste(t)

	if imagem.Image == nil {
		t.Fatal("preview inicial está vazio")
	}
	if imagem.FillMode != canvas.ImageFillContain {
		t.Errorf("FillMode = %v, esperado ImageFillContain", imagem.FillMode)
	}

	assertMesmaImagem(t, r.render(signatureData{}), imagem.Image)
}

// TestPreviewAtualizaAoDigitar é o requisito central da mudança.
func TestPreviewAtualizaAoDigitar(t *testing.T) {
	imagem, nome, r := montarPreviewParaTeste(t)
	inicial := imagem.Image

	nome.SetText("Fabiano Chiaretto Fernandes")

	if imagem.Image == inicial {
		t.Fatal("preview não foi atualizado após digitar")
	}
	assertMesmaImagem(t, r.render(signatureData{name: "Fabiano Chiaretto Fernandes"}), imagem.Image)
}

// TestPreviewVoltaAoLimparCampo garante que apagar o texto realmente o remove
// da imagem, em vez de deixar o desenho anterior grudado no modelo em cache.
func TestPreviewVoltaAoLimparCampo(t *testing.T) {
	imagem, nome, r := montarPreviewParaTeste(t)

	nome.SetText("Fulano de Tal")
	nome.SetText("")

	assertMesmaImagem(t, r.render(signatureData{}), imagem.Image)
}

// TestPreviewCorrespondeAoArquivoSalvo liga as duas pontas: o que está na tela
// é exatamente o que vai para o disco.
func TestPreviewCorrespondeAoArquivoSalvo(t *testing.T) {
	imagem, nome, r := montarPreviewParaTeste(t)

	nome.SetText("Fabiano Chiaretto Fernandes")

	destino := t.TempDir() + "/assinatura.png"
	if err := r.save(signatureData{name: "Fabiano Chiaretto Fernandes"}, destino); err != nil {
		t.Fatalf("save: %v", err)
	}

	gravada, err := gg.LoadPNG(destino)
	if err != nil {
		t.Fatalf("erro ao reler o arquivo gravado: %v", err)
	}

	assertMesmaImagem(t, gravada, imagem.Image)
}

// TestPreviewRemoveTracoAoApagarRegistro cobre o cenário do usuário: com o
// preview aberto, apagar o registro tem de tirar o traço da tela na hora.
func TestPreviewRemoveTracoAoApagarRegistro(t *testing.T) {
	test.NewApp()

	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}

	profissao := widget.NewEntry()
	profissao.SetText("Advogado")
	registro := widget.NewEntry()
	registro.SetText("OAB/MG - 123456")

	coletar := func() signatureData {
		return signatureData{profession: profissao.Text, registration: registro.Text}
	}

	obj := montarPreview(r, nil, []*widget.Entry{profissao, registro}, coletar,
		widget.NewButton("gerar", func() {}))
	imagem, ok := obj.(*canvas.Image)
	if !ok {
		t.Fatalf("esperava *canvas.Image, obtive %T", obj)
	}

	// Antes: com registro, o traço aparece.
	assertMesmaImagem(t,
		renderRef(t, signatureData{}, "Advogado - OAB/MG - 123456"),
		imagem.Image)

	registro.SetText("")

	// Depois: só a profissão, sem traço órfão.
	assertMesmaImagem(t, renderRef(t, signatureData{}, "Advogado"), imagem.Image)
}

// TestModoDegradado cobre o caso de assets ausentes: a janela precisa continuar
// utilizável, sem panic, com a geração desabilitada.
func TestModoDegradado(t *testing.T) {
	test.NewApp()

	botao := widget.NewButton("gerar", func() {})
	obj := montarPreview(nil, errors.New("modelo-assinatura.png: no such file"), nil, nil, botao)

	if obj == nil {
		t.Fatal("montarPreview devolveu nil em vez de uma mensagem de erro")
	}
	if _, ok := obj.(*canvas.Image); ok {
		t.Error("em modo degradado não deveria haver imagem de preview")
	}
	if !botao.Disabled() {
		t.Error("o botão de gerar deveria estar desabilitado")
	}
}
