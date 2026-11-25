# Sample data — synthetic

**Not real companies.** Sixty made-up establishments, with valid CNPJ check
digits, in the layout of the Receita Federal's public CNPJ data, so `seed`
runs without downloading anything:

- `ESTABELECIMENTOS.csv` — the *Estabelecimentos* layout: `;`-separated,
  quoted, ISO-8859-1, 30 fields per row.
- `EMPRESAS.csv` — the *Empresas* layout: 7 fields per row.

For the real data, download the monthly dump from the Receita Federal
(*Dados Abertos do CNPJ*, `arquivos.receitafederal.gov.br`), unzip one
`Estabelecimentos` file and one `Empresas` file, and pass them to `seed`
with `-estabelecimentos` and `-empresas`. Only active establishments
(situação `02`) become entities.
