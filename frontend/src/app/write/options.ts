/**
 * The choices the composer offers: identifier kinds, languages, ratings.
 */

// --- IMPORTS ---
import type { KindOption } from '@/app/write/Write.t'
import type { Rating } from '@/logic/protocol/review.t'

// --- GLOBALS ---
export const KINDS: KindOption[] = [
  {
    value: 'cnpj',
    label: 'CNPJ',
    placeholder: '12.345.678/0001-99',
    hint: 'A company, by its CNPJ. Punctuation is optional.',
  },
  {
    value: 'domain',
    label: 'Website',
    placeholder: 'example.com.br',
    hint: 'A website, by its address. No https:// or path needed.',
  },
  {
    value: 'gtin',
    label: 'Product',
    placeholder: '7891234567895',
    hint: 'A product, by the barcode number on its package (EAN or GTIN).',
  },
]

export const LANGUAGES = [
  { value: '', label: 'Language not set' },
  { value: 'pt-BR', label: 'Portuguese (Brazil)' },
  { value: 'en', label: 'English' },
  { value: 'es', label: 'Spanish' },
]

export const RATINGS: Rating[] = [1, 2, 3, 4, 5]
