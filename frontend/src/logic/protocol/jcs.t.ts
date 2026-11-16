/**
 * Types for canonicalization.
 */

// --- GLOBALS ---
/** A JSON object; a key left undefined is as good as absent. */
export type JsonObject = { [key: string]: Json | undefined }

/** Any value canonicalization accepts: no floats, no undefined in arrays. */
export type Json =
  | null
  | boolean
  | number
  | string
  | Json[]
  | JsonObject
