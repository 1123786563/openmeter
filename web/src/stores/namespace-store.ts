import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface NamespaceState {
  /**
   * Explicitly selected namespace, or null to ride on the server default.
   * `null` is the initial state; the switcher falls back to the namespace
   * list's `default` value once loaded.
   */
  currentNamespace: string | null
  setNamespace: (namespace: string | null) => void
}

export const useNamespaceStore = create<NamespaceState>()(
  persist(
    (set) => ({
      currentNamespace: null,
      setNamespace: (namespace) => set({ currentNamespace: namespace }),
    }),
    { name: 'openmeter-admin.namespace' }
  )
)
