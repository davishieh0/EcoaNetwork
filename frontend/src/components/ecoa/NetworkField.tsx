/**
 * A network drifting behind a page: peers joined by thin edges, and now and
 * then an event travelling from one to the next, as signed events spread.
 */

// --- IMPORTS ---
import { useEffect, useRef } from 'react'
import type {
  FieldColors,
  FieldNode,
  NetworkFieldProps,
  Pulse,
  Ring,
} from '@/components/ecoa/NetworkField.t'
import { cn } from '@/components/ui/cn'

// --- GLOBALS ---
// one node for this many square pixels
const AREA_PER_NODE = 16000

// each node joins its nearest neighbours, never farther than this
const NEIGHBOURS = 2
const REACH = 240

// depth sets back through opacity and drift, as the landing's field does
const DEPTH_OPACITY = [1, 0.66, 0.4] as const
const DRIFT = [9, 6, 3.5] as const

// how often an event sets off, and how far a chain of hops may go
const LAUNCH_EVERY = 1.1
const MOST_HOPS = 4
const CARRY_ON = 0.6

// the lit stretch that travels along an edge, in pixels
const TRAIL = 70

const RING_SECONDS = 0.9

// the pointer lights the nodes within this distance
const LANTERN = 150

// behind the page's column of text the field steps back to this much, so
// it lives at the sides and never under a line someone is reading
const QUIET = 0.25
const COLUMN = 470
const EASE_IN = 80

// --- CODE ---
/**
 * A small deterministic random source, so the field is the same on every
 * visit for the same size.
 *
 * @param seed - Where the sequence starts.
 * @returns The next number in [0, 1) at each call.
 */
function seeded(seed: number) {
  let state = seed >>> 0

  return () => {
    state = (state + 0x6d2b79f5) >>> 0
    let t = state
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

/**
 * Lays out nodes over an area, spread on a jittered grid so none clump.
 *
 * @param width - The area's width, in CSS pixels.
 * @param height - Its height.
 * @returns The nodes.
 */
function layout(width: number, height: number): FieldNode[] {
  const random = seeded(Math.round(width / 50) * 7919)
  const count = Math.max(12, Math.round((width * height) / AREA_PER_NODE))
  const columns = Math.ceil(Math.sqrt((count * width) / height))
  const rows = Math.ceil(count / columns)
  const cellW = width / columns
  const cellH = height / rows

  const nodes: FieldNode[] = []
  for (let row = 0; row < rows; row++) {
    for (let column = 0; column < columns; column++) {
      const event = random() < 0.12
      const depth = (random() < 0.25 ? 0 : random() < 0.55 ? 1 : 2) as
        | 0
        | 1
        | 2

      nodes.push({
        x: (column + 0.15 + random() * 0.7) * cellW,
        y: (row + 0.15 + random() * 0.7) * cellH,
        r: event ? 3 + random() * 2 : 2.5 + (2 - depth) * 1.2 + random(),
        kind: event ? 'event' : 'actor',
        depth,
        phase: random() * Math.PI * 2,
        period: 11 + random() * 9,
      })
    }
  }

  return nodes
}

/**
 * Joins each node to its nearest neighbours within reach.
 *
 * @param nodes - The nodes.
 * @returns The edges, as index pairs, each once.
 */
function join(nodes: FieldNode[]): [number, number][] {
  const seen = new Set<string>()
  const edges: [number, number][] = []

  nodes.forEach((node, index) => {
    nodes
      .map((other, at) => ({
        at,
        d: Math.hypot(other.x - node.x, other.y - node.y),
      }))
      .filter(({ at, d }) => at !== index && d < REACH)
      .sort((a, b) => a.d - b.d)
      .slice(0, NEIGHBOURS)
      .forEach(({ at }) => {
        const key = index < at ? `${index}-${at}` : `${at}-${index}`
        if (seen.has(key)) return
        seen.add(key)
        edges.push([index, at])
      })
  })

  return edges
}

/**
 * Reads the colour tokens the field draws with.
 *
 * @param element - Any element under the page's theme.
 * @returns The colours.
 */
function colorsOf(element: Element): FieldColors {
  const style = getComputedStyle(element)
  const token = (name: string) => style.getPropertyValue(name).trim()

  return {
    signal: token('--ecoa-signal') || '#6d9bff',
    line: token('--ecoa-border-strong') || 'rgba(255,255,255,0.34)',
    node: token('--ecoa-border-structural') || '#68707b',
    canvas: token('--ecoa-canvas') || '#0e0f10',
  }
}

/**
 * Draws the network on a canvas filling its parent.
 *
 * @param props - Extra classes.
 * @returns The canvas, hidden from assistive technology.
 */
export function NetworkField({ className }: NetworkFieldProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    const canvas = canvasRef.current
    const context = canvas?.getContext('2d')
    if (!canvas || !context) return undefined

    const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    let still = motionQuery.matches

    let width = 0
    let height = 0
    let nodes: FieldNode[] = []
    let edges: [number, number][] = []
    let neighbours: number[][] = []
    let colors = colorsOf(canvas)

    let pulses: Pulse[] = []
    let rings: Ring[] = []
    let sinceLaunch = LAUNCH_EVERY * 0.6
    let clock = 0
    let last = 0
    let frame = 0
    let visible = true
    const pointer = { x: -9999, y: -9999 }

    const random = seeded(42)

    // where a node is now, drifting about where it was laid
    const at = (index: number) => {
      const node = nodes[index]
      if (still) return { x: node.x, y: node.y }
      const turn = (clock / node.period) * Math.PI * 2 + node.phase
      return {
        x: node.x + Math.sin(turn) * DRIFT[node.depth],
        y: node.y + Math.cos(turn * 0.8 + 1.3) * DRIFT[node.depth],
      }
    }

    const resize = () => {
      const rect = canvas.getBoundingClientRect()
      const dpr = Math.min(window.devicePixelRatio || 1, 2)
      width = rect.width
      height = rect.height
      canvas.width = Math.round(width * dpr)
      canvas.height = Math.round(height * dpr)
      context.setTransform(dpr, 0, 0, dpr, 0, 0)

      nodes = layout(width, height)
      edges = join(nodes)
      neighbours = nodes.map(() => [])
      edges.forEach(([a, b]) => {
        neighbours[a].push(b)
        neighbours[b].push(a)
      })
      pulses = []
      rings = []
      draw()
    }

    // an event sets off from a node to one of its neighbours
    const launch = (from: number, hop: number) => {
      const options = neighbours[from]
      if (!options?.length) return
      const to = options[Math.floor(random() * options.length)]
      pulses.push({ from, to, age: 0, duration: 1.6 + random() * 1.2, hop })
    }

    // how present the field is at x: quiet behind the text, full aside
    const presence = (x: number) => {
      const half = Math.min(width * 0.36, COLUMN)
      const off = Math.abs(x - width / 2) - half
      const t = Math.min(Math.max((off + EASE_IN) / (EASE_IN * 2), 0), 1)
      return QUIET + (1 - QUIET) * t
    }

    const draw = () => {
      context.clearRect(0, 0, width, height)
      const places = nodes.map((_, index) => at(index))

      // the edges, dashed and faint
      context.setLineDash([3, 6])
      context.lineWidth = 1
      context.strokeStyle = colors.line
      for (const [a, b] of edges) {
        const depth = Math.max(nodes[a].depth, nodes[b].depth)
        const middle = (places[a].x + places[b].x) / 2
        context.globalAlpha = 0.5 * DEPTH_OPACITY[depth] * presence(middle)
        context.beginPath()
        context.moveTo(places[a].x, places[a].y)
        context.lineTo(places[b].x, places[b].y)
        context.stroke()
      }

      // the events on their way: a lit stretch riding the edge
      context.strokeStyle = colors.signal
      context.lineWidth = 1.5
      for (const pulse of pulses) {
        const a = places[pulse.from]
        const b = places[pulse.to]
        const length = Math.hypot(b.x - a.x, b.y - a.y) || 1
        const progress = Math.min(pulse.age / pulse.duration, 1)
        const head = progress * (length + TRAIL)
        const start = Math.max(0, head - TRAIL) / length
        const end = Math.min(head, length) / length

        // an event stays visible behind the text, only softer
        const x = a.x + (b.x - a.x) * end
        context.globalAlpha = 0.9 * Math.max(presence(x), 0.45)
        context.beginPath()
        context.moveTo(a.x + (b.x - a.x) * start, a.y + (b.y - a.y) * start)
        context.lineTo(a.x + (b.x - a.x) * end, a.y + (b.y - a.y) * end)
        context.stroke()
      }
      context.setLineDash([])

      // the rings where events arrived
      for (const ring of rings) {
        const place = places[ring.node]
        const progress = ring.age / RING_SECONDS
        context.globalAlpha =
          0.7 * (1 - progress) * Math.max(presence(place.x), 0.45)
        context.lineWidth = 1.5
        context.beginPath()
        context.arc(
          place.x,
          place.y,
          nodes[ring.node].r * (1 + progress * 2.4),
          0,
          Math.PI * 2,
        )
        context.stroke()
      }

      // the nodes, lit near the pointer
      for (let index = 0; index < nodes.length; index++) {
        const node = nodes[index]
        const place = places[index]
        const near = Math.max(
          0,
          1 - Math.hypot(place.x - pointer.x, place.y - pointer.y) / LANTERN,
        )
        const opacity = Math.min(
          1,
          DEPTH_OPACITY[node.depth] * presence(place.x) + near * 0.6,
        )

        context.globalAlpha = opacity
        if (node.kind === 'event') {
          context.fillStyle = colors.signal
          const r = node.r * (1 + near * 0.3)
          context.fillRect(place.x - r, place.y - r, r * 2, r * 2)
        } else {
          context.fillStyle = colors.canvas
          context.strokeStyle = near > 0.2 ? colors.signal : colors.node
          context.lineWidth = 1.5
          context.beginPath()
          const r = node.r * (1 + near * 0.3)
          context.arc(place.x, place.y, r, 0, Math.PI * 2)
          context.fill()
          context.stroke()
        }
      }

      context.globalAlpha = 1
    }

    const tick = (now: number) => {
      const step = last ? Math.min((now - last) / 1000, 0.05) : 0
      last = now
      clock += step

      // set events off, and move the ones on their way
      sinceLaunch += step
      if (sinceLaunch >= LAUNCH_EVERY && nodes.length) {
        sinceLaunch = 0
        launch(Math.floor(random() * nodes.length), 0)
      }

      const arrived: Pulse[] = []
      pulses = pulses.filter((pulse) => {
        pulse.age += step
        if (pulse.age < pulse.duration) return true
        arrived.push(pulse)
        return false
      })
      for (const pulse of arrived) {
        rings.push({ node: pulse.to, age: 0 })
        // the node it reached passes it on, now and then
        if (pulse.hop < MOST_HOPS && random() < CARRY_ON) {
          launch(pulse.to, pulse.hop + 1)
        }
      }
      rings = rings.filter((ring) => (ring.age += step) < RING_SECONDS)

      draw()
      frame = requestAnimationFrame(tick)
    }

    const start = () => {
      cancelAnimationFrame(frame)
      last = 0
      if (!still && visible && !document.hidden) {
        frame = requestAnimationFrame(tick)
      } else {
        draw()
      }
    }

    const onPointerMove = (event: PointerEvent) => {
      const rect = canvas.getBoundingClientRect()
      pointer.x = event.clientX - rect.left
      pointer.y = event.clientY - rect.top
      if (still) draw()
    }

    const onMotionChange = () => {
      still = motionQuery.matches
      start()
    }

    // the theme can change under the page: read the tokens again
    const onThemeChange = () => {
      colors = colorsOf(canvas)
      draw()
    }

    const sizes = new ResizeObserver(resize)
    sizes.observe(canvas)

    // nothing is drawn while the band is off screen
    const sight = new IntersectionObserver(([entry]) => {
      visible = entry.isIntersecting
      start()
    })
    sight.observe(canvas)

    const themes = new MutationObserver(onThemeChange)
    themes.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-theme', 'class'],
    })

    window.addEventListener('pointermove', onPointerMove, { passive: true })
    document.addEventListener('visibilitychange', start)
    motionQuery.addEventListener('change', onMotionChange)

    resize()
    start()

    return () => {
      cancelAnimationFrame(frame)
      sizes.disconnect()
      sight.disconnect()
      themes.disconnect()
      window.removeEventListener('pointermove', onPointerMove)
      document.removeEventListener('visibilitychange', start)
      motionQuery.removeEventListener('change', onMotionChange)
    }
  }, [])

  return (
    <canvas
      ref={canvasRef}
      aria-hidden="true"
      className={cn('block h-full w-full', className)}
    />
  )
}
