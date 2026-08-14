package main

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogleman/gg"
)

var amostra = signatureData{
	name:         "Fabiano Chiaretto Fernandes",
	email:        "fabiano@hfcd.com.br",
	profession:   "Advogado",
	registration: "OAB/MG - 123456",
	telefones:    "(34) 9 9126-5991 / (34) 3232-6699",
}

// renderRef é o oráculo independente dos testes: desenha a assinatura do jeito
// original — recarregando o modelo e reparseando as fontes a cada chamada —
// mas recebe a linha de profissão já pronta, em vez de montá-la. Isso permite
// afirmar exatamente qual texto deveria aparecer, sem depender da função que
// está sendo testada.
func renderRef(tb testing.TB, d signatureData, linhaProfissao string) image.Image {
	tb.Helper()

	bgImage, err := gg.LoadPNG("modelo-assinatura.png")
	if err != nil {
		tb.Fatalf("erro ao carregar a imagem de fundo: %v", err)
	}

	dc := gg.NewContextForImage(bgImage)
	dc.SetHexColor("#4C4C4C")

	if err := dc.LoadFontFace("./Arial.ttf", 40); err != nil {
		tb.Fatalf("erro ao carregar a fonte: %v", err)
	}
	dc.DrawStringAnchored(d.name, 300, 162, 0, 0)

	if err := dc.LoadFontFace("./Arial.ttf", 30); err != nil {
		tb.Fatalf("erro ao carregar a fonte: %v", err)
	}
	dc.DrawStringAnchored(linhaProfissao, 300, 198, 0, 0)
	dc.DrawStringAnchored(d.email, 300, 265, 0, 0)

	if err := dc.LoadFontFace("./ArialBold.ttf", 25); err != nil {
		tb.Fatalf("erro ao carregar a fonte: %v", err)
	}
	dc.DrawStringAnchored(d.telefones, 900, 148, 0, 0)

	return dc.Image()
}

// legacyRender reproduz a implementação anterior ao cache de assets, incluindo
// a concatenação antiga `profissão + " - " + registro`. Serve de referência
// para o refactor de desempenho, que não podia mudar um único pixel.
//
// Ressalva: depois que o traço passou a ser omitido para registro vazio, ela só
// continua equivalente quando o registro está preenchido — que é o caso de
// `amostra`.
func legacyRender(tb testing.TB, d signatureData) image.Image {
	tb.Helper()
	return renderRef(tb, d, d.profession+" - "+d.registration)
}

// primeiroPixelDiferente devolve as coordenadas da primeira divergência entre
// as duas imagens, ou ok = false se forem idênticas.
func primeiroPixelDiferente(a, b image.Image) (x, y int, ok bool) {
	if a.Bounds() != b.Bounds() {
		return 0, 0, true
	}

	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.At(x, y) != b.At(x, y) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func assertMesmaImagem(t *testing.T, esperado, obtido image.Image) {
	t.Helper()

	if esperado.Bounds() != obtido.Bounds() {
		t.Fatalf("bounds divergiram: esperado %v, obtido %v", esperado.Bounds(), obtido.Bounds())
	}

	if x, y, diferente := primeiroPixelDiferente(esperado, obtido); diferente {
		t.Fatalf("pixel (%d,%d) divergiu: esperado %v, obtido %v",
			x, y, esperado.At(x, y), obtido.At(x, y))
	}
}

// TestRenderEquivaleAImplementacaoAnterior é o teste de regressão que sustenta
// todo o refactor de desempenho. `amostra` tem registro preenchido, que é o
// caso em que a formatação não mudou.
func TestRenderEquivaleAImplementacaoAnterior(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}

	assertMesmaImagem(t, legacyRender(t, amostra), r.render(amostra))
}

// TestRenderLinhaProfissao verifica, no pixel, qual texto vai para a linha de
// profissão em cada combinação de preenchimento. O oráculo é renderRef, que
// recebe o texto esperado literalmente.
func TestRenderLinhaProfissao(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}

	casos := []struct {
		nome       string
		profession string
		registro   string
		esperado   string
	}{
		{"ambos preenchidos", "Advogado", "OAB/MG - 123456", "Advogado - OAB/MG - 123456"},
		{"sem registro", "Advogado", "", "Advogado"},
		{"registro só com espaços", "Advogado", "   ", "Advogado"},
		{"sem profissão", "", "OAB/MG - 123456", "OAB/MG - 123456"},
		{"ambos vazios", "", "", ""},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			d := amostra
			d.profession = c.profession
			d.registration = c.registro

			assertMesmaImagem(t, renderRef(t, d, c.esperado), r.render(d))
		})
	}
}

// TestRenderSemRegistroDivergeDoComportamentoAntigo é a prova de que a mudança
// realmente saiu do papel: com registro vazio, a imagem não pode mais ser igual
// à que o código antigo produzia (que terminava em "Advogado - ").
func TestRenderSemRegistroDivergeDoComportamentoAntigo(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}

	semRegistro := amostra
	semRegistro.registration = ""

	if _, _, diferente := primeiroPixelDiferente(legacyRender(t, semRegistro), r.render(semRegistro)); !diferente {
		t.Error("imagem idêntica à do comportamento antigo — o traço órfão continua sendo desenhado")
	}
}

// TestRenderNaoReaproveitaBuffer garante que uma renderização não corrompe a
// imagem devolvida pela anterior — pré-requisito para o preview em tempo real,
// que retém a imagem enquanto a próxima é composta.
func TestRenderNaoReaproveitaBuffer(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}

	primeira := r.render(amostra)

	outra := amostra
	outra.name = "Outro Nome Completamente Diferente"
	r.render(outra)

	assertMesmaImagem(t, legacyRender(t, amostra), primeira)
}

// TestRenderEstavel confirma que renderizações sucessivas com os mesmos dados
// produzem o mesmo resultado, ou seja, que o modelo em cache não acumula os
// textos desenhados.
func TestRenderEstavel(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}

	primeira := r.render(amostra)
	for i := 0; i < 3; i++ {
		r.render(amostra)
	}

	assertMesmaImagem(t, primeira, r.render(amostra))
}

func TestSaveGravaOMesmoQueRender(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}

	destino := filepath.Join(t.TempDir(), "assinatura.png")
	if err := r.save(amostra, destino); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := os.Stat(destino); err != nil {
		t.Fatalf("arquivo não foi criado: %v", err)
	}

	gravada, err := gg.LoadPNG(destino)
	if err != nil {
		t.Fatalf("erro ao reler o arquivo gravado: %v", err)
	}

	assertMesmaImagem(t, r.render(amostra), gravada)
}

func TestJuntarProfissao(t *testing.T) {
	casos := []struct {
		nome       string
		profession string
		registro   string
		esperado   string
	}{
		{"ambos preenchidos", "Advogado", "OAB/MG - 123456", "Advogado - OAB/MG - 123456"},
		{"sem registro", "Advogado", "", "Advogado"},
		{"registro só com espaços", "Advogado", "   ", "Advogado"},
		{"sem profissão", "", "OAB/MG - 123456", "OAB/MG - 123456"},
		{"ambos vazios", "", "", ""},
		{"espaços ao redor", "  Advogado ", " OAB/MG - 123456  ", "Advogado - OAB/MG - 123456"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if obtido := juntarProfissao(c.profession, c.registro); obtido != c.esperado {
				t.Errorf("juntarProfissao(%q, %q) = %q, esperado %q",
					c.profession, c.registro, obtido, c.esperado)
			}
		})
	}
}

func TestJuntarTelefones(t *testing.T) {
	casos := []struct {
		nome     string
		celular  string
		telefone string
		esperado string
	}{
		{"ambos preenchidos", "(34) 9 9126-5991", "(34) 3232-6699", "(34) 9 9126-5991 / (34) 3232-6699"},
		{"sem celular", "", "(34) 3232-6699", "(34) 3232-6699"},
		{"sem telefone", "(34) 9 9126-5991", "", "(34) 9 9126-5991"},
		{"ambos vazios", "", "", ""},
		{"apenas espaços", "   ", "  ", ""},
		{"espaços ao redor", "  (34) 9 9126-5991  ", "  (34) 3232-6699 ", "(34) 9 9126-5991 / (34) 3232-6699"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if obtido := juntarTelefones(c.celular, c.telefone); obtido != c.esperado {
				t.Errorf("juntarTelefones(%q, %q) = %q, esperado %q", c.celular, c.telefone, obtido, c.esperado)
			}
		})
	}
}

// BenchmarkRender mede uma renderização com os assets já em cache — o número
// que precisa caber no orçamento de uma tecla digitada.
func BenchmarkRender(b *testing.B) {
	r, err := newRenderer()
	if err != nil {
		b.Fatalf("newRenderer: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.render(amostra)
	}
}

// BenchmarkLegacyRender mede o caminho anterior, com o modelo e as fontes
// recarregados do disco a cada chamada. A diferença entre os dois é o ganho
// do refactor.
func BenchmarkLegacyRender(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = legacyRender(b, amostra)
	}
}
