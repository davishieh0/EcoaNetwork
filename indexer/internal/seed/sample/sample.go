// Package sample is a synthetic stand-in for the Receita Federal's CNPJ
// data, in its layout: sixty made-up establishments, so seed runs without
// downloading anything. See README.md.
package sample


import (
	"bytes"
	_ "embed"
	"io"
)


// --- GLOBALS ---

//go:embed ESTABELECIMENTOS.csv
var estabelecimentos []byte

//go:embed EMPRESAS.csv
var empresas []byte


// --- CODE ---

// Estabelecimentos returns the sample's Estabelecimentos file.
func Estabelecimentos() io.Reader {
	return bytes.NewReader(estabelecimentos)
}


// Empresas returns the sample's Empresas file.
func Empresas() io.Reader {
	return bytes.NewReader(empresas)
}
