/**
 * Types for the GradientASCII backdrop.
 */

// --- IMPORTS ---
import type { CSSProperties, HTMLAttributes } from 'react'

// --- GLOBALS ---
/** Gradient field shape preset. */
export type GradientASCIIShape = 'horizon' | 'pinch' | 'rise' | 'eclipse'

/** Interactive ripple style triggered on pointer down. */
export type GradientASCIIClickEffect = 'wave' | 'burst' | 'none'

/** Props accepted by the GradientASCII backdrop component. */
export interface GradientASCIIProps extends Omit<
  HTMLAttributes<HTMLDivElement>,
  'style'
> {
  colors?: string[]
  shape?: GradientASCIIShape
  characters?: string
  cellSize?: number
  density?: number
  glyphOpacity?: number
  glow?: number
  depthOfField?: number
  flicker?: number
  softness?: number
  scale?: number
  flow?: number
  speed?: number
  gradientOpacity?: number
  grain?: number
  variation?: number
  mouseInteraction?: boolean
  scrambleRadius?: number
  clickEffect?: GradientASCIIClickEffect
  intro?: boolean
  lightMode?: boolean
  fontFamily?: string
  dpr?: number
  style?: CSSProperties
}

/** Normalized runtime settings read by the WebGL2 render loop. */
export type Settings = {
  colors: string[]
  shape: number
  characters: string
  cellSize: number
  density: number
  glyphOpacity: number
  glow: number
  depthOfField: number
  flicker: number
  softness: number
  scale: number
  flow: number
  speed: number
  gradientOpacity: number
  grain: number
  variation: number
  mouseInteraction: boolean
  scrambleRadius: number
  clickEffect: number
  intro: boolean
  lightMode: boolean
  fontFamily: string
  dpr?: number
}

/** Linked WebGL2 program paired with its resolved uniform locations. */
export type Program = {
  program: WebGLProgram
  uniforms: Record<string, WebGLUniformLocation | null>
}

/** Offscreen framebuffer and its color attachment textures. */
export type Target = {
  textures: (WebGLTexture | null)[]
  framebuffer: WebGLFramebuffer | null
  width: number
  height: number
}

/** Rendered glyph atlas texture and its cell grid dimensions. */
export type Atlas = {
  texture: WebGLTexture | null
  count: number
  grid: number[]
}

/** Pointer trail or click impulse sample in normalized aspect space. */
export type Point = {
  x: number
  y: number
  age: number
}
