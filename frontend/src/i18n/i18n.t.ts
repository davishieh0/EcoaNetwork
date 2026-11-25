/**
 * Types for the interface's languages.
 */

// --- GLOBALS ---
/** A language the interface speaks. */
export type Language = 'en' | 'pt-BR'

/** The translate function a component gets from useTranslation. */
export type Translate = (key: string) => string
