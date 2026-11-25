/**
 * Inspecting one signed event: reading it alone and checking it here.
 */

// --- IMPORTS ---
import { useCallback, useEffect, useMemo, useState } from 'react'
import { messageOf } from '@/app/forms/messages'
import { useRead } from '@/app/reads/useRead'
import { momentOf } from '@/app/reads/words'
import type {
  Check,
  EventInspectorProps,
  InspectedEvent,
} from '@/components/ecoa/EventInspector.t'
import { getEvent } from '@/logic/datasource/events'
import type { EventItem } from '@/logic/datasource/events.t'
import { readEnvelope } from '@/logic/protocol/signed'
import { verifyEnvelope } from '@/logic/protocol/verify'
import type { Verdict } from '@/logic/protocol/verify.t'

// --- GLOBALS ---
const NOTES = [
  'Signed at is the author’s clock, inside the signature; received is when ' +
    'the network first had it, as this index observed.',
  'Which nodes hold it is not tracked by this index yet.',
]

// --- CODE ---
/**
 * Lays out what an event says and what was observed about it.
 *
 * @param item - The event, as the index answered it.
 * @param check - Where checking its signature is.
 * @returns The event, as the inspector shows it.
 */
function inspectedOf(item: EventItem, check: Check): InspectedEvent {
  const envelope = readEnvelope(item.envelope)

  const rows = [
    { label: 'Event ID', value: envelope.eventId },
    { label: 'Type', value: envelope.eventType },
    { label: 'Author', value: envelope.author },
    ...(envelope.deviceKey
      ? [{ label: 'Device key', value: String(envelope.deviceKey) }]
      : []),
    { label: 'Signed at', value: momentOf(envelope.createdAt) },
    ...(item.observed.receivedAt
      ? [{ label: 'Received', value: momentOf(item.observed.receivedAt) }]
      : []),
    { label: 'Network', value: envelope.protocol },
    { label: 'Signature', value: envelope.signature },
  ]

  return {
    rows,
    check,
    badges: item.observed.labels?.map((applied) => ({
      label: `Labeled · ${applied.label}`,
      tone: 'warning',
      glyph: '●',
    })),
    raw: item.envelope,
    notes: NOTES,
  }
}

/**
 * Holds which event is open, reads it, and checks its signature.
 *
 * @returns inspect, which opens an event, and the inspector's props.
 */
export function useInspection() {
  const [eventId, setEventId] = useState<string | null>(null)

  const read = useCallback(
    () => (eventId ? getEvent(eventId) : Promise.resolve(null)),
    [eventId],
  )
  const { data, error } = useRead(read)

  // the verdict, with the envelope it is about
  const [checked, setChecked] = useState<{
    raw: string
    verdict: Verdict
  } | null>(null)

  useEffect(() => {
    if (!data) return
    let live = true

    verifyEnvelope(data.envelope).then(
      (verdict) => live && setChecked({ raw: data.envelope, verdict }),
    )

    return () => {
      live = false
    }
  }, [data])

  const event = useMemo(() => {
    if (!data) return null

    const verdict = checked?.raw === data.envelope ? checked.verdict : null
    const check: Check = !verdict
      ? { state: 'checking' }
      : verdict.ok === true
        ? { state: 'valid' }
        : verdict.ok === null
          ? { state: 'unchecked', reason: verdict.reason }
          : { state: 'invalid', reason: verdict.reason }

    return inspectedOf(data, check)
  }, [data, checked])

  const inspector: EventInspectorProps = {
    open: eventId !== null,
    onOpenChange: (open) => {
      if (!open) setEventId(null)
    },
    event,
    error: error ? messageOf(error) : undefined,
  }

  return { inspect: setEventId, inspector }
}
