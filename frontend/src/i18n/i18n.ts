/**
 * The interface's languages. English is the source and the default: each
 * English string is its own key, and pt-BR maps it to Portuguese.
 */

// --- IMPORTS ---
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import type { Language, Translate } from '@/i18n/i18n.t'
import app from '@/i18n/locales/pt-BR/app.json'
import site from '@/i18n/locales/pt-BR/site.json'

// --- GLOBALS ---
/** Every language the interface speaks, the default first. */
export const LANGUAGES: readonly Language[] = ['en', 'pt-BR']

const STORAGE_KEY = 'ecoa.language'

const TITLE = 'Ecoa — Every experience leaves an echo'

const DESCRIPTION =
  'Ecoa is an open network of verifiable experiences and reviews. Signed ' +
  'by their authors, readable by anyone, owned by no single platform.'

// --- CODE ---
/**
 * Reads the language this reader chose before, if any.
 *
 * @returns The stored language, or English.
 */
function storedLanguage(): Language {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'pt-BR' ? 'pt-BR' : 'en'
  } catch {
    return 'en'
  }
}

/**
 * Switches the interface to a language and remembers the choice.
 *
 * @param language - The language to speak.
 * @returns Nothing.
 */
export function setLanguage(language: Language) {
  try {
    localStorage.setItem(STORAGE_KEY, language)
  } catch {
    // a browser that keeps nothing still switches, for this visit
  }

  void i18n.changeLanguage(language)
}

/**
 * Tells which language the interface speaks now.
 *
 * @returns The current language.
 */
export function currentLanguage(): Language {
  return i18n.language === 'pt-BR' ? 'pt-BR' : 'en'
}

/**
 * Translates copy the dictionary knows, and leaves any other text as it is:
 * text already worded, or carrying what a person typed, is never parsed.
 *
 * @param t - The translate function of the calling component.
 * @param text - English copy, or text that is already final.
 * @returns The translation, or the text unchanged.
 */
export function known(t: Translate, text: string): string {
  return i18n.exists(text) ? t(text) : text
}

/**
 * Keeps the document's language, title and description in step.
 *
 * @param language - The language just set.
 * @returns Nothing.
 */
function syncDocument(language: string) {
  document.documentElement.lang = language
  document.title = i18n.t(TITLE)
  document
    .querySelector('meta[name="description"]')
    ?.setAttribute('content', i18n.t(DESCRIPTION))
}

i18n.on('languageChanged', syncDocument)

void i18n.use(initReactI18next).init({
  lng: storedLanguage(),
  fallbackLng: 'en',
  supportedLngs: [...LANGUAGES],
  resources: { 'pt-BR': { translation: { ...site, ...app } } },
  // NOTE: keys are English sentences, which carry '.' and ':'; with the
  // separators on, i18next would split them into paths.
  keySeparator: false,
  nsSeparator: false,
  interpolation: { escapeValue: false },
  // a phrase pt-BR words elsewhere is translated as nothing, on purpose
  returnEmptyString: true,
  react: { useSuspense: false },
})

export { i18n }
