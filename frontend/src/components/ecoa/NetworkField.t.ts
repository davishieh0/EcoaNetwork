/**
 * Types for the NetworkField.
 */

// --- GLOBALS ---
/** A point of the network: a peer, or an event on its way. */
export type FieldNode = {
  x: number
  y: number
  r: number
  kind: 'actor' | 'event'
  /** 0 is nearest; deeper ones sit back, fainter and slower. */
  depth: 0 | 1 | 2
  phase: number
  period: number
}

/** An event travelling along an edge, from one node to the next. */
export type Pulse = {
  from: number
  to: number
  /** Seconds since it left. */
  age: number
  duration: number
  /** How many hops it has made, so a chain dies out. */
  hop: number
}

/** The ring a pulse opens on the node it reaches. */
export type Ring = {
  node: number
  age: number
}

/** The tokens the field draws with, read from the page. */
export type FieldColors = {
  signal: string
  line: string
  node: string
  canvas: string
}

export type NetworkFieldProps = {
  className?: string
}
