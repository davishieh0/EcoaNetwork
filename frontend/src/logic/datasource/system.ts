/**
 * Observed network nodes, as the indexer reports them from its cursors.
 */

// --- IMPORTS ---
import { request } from '@/logic/datasource/http'

// --- GLOBALS ---
export type ObservedNode = {
  node: string
  logId: string
  sequence: number
  events: number
  lastReceivedAt?: string
}

export type ObservedNodesResponse = {
  nodes: ObservedNode[]
}

// --- CODE ---
/**
 * Reads the nodes the indexer follows and their observed log positions.
 *
 * @returns The list of observed nodes.
 */
export function getObservedNodes() {
  return request<ObservedNodesResponse>('/system/nodes')
}
