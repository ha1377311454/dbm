import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api'
import type { ConnectionConfig, Group } from '@/types'

export const useConnectionsStore = defineStore('connections', () => {
  const connections = ref<ConnectionConfig[]>([])
  const groups = ref<Group[]>([])
  const loading = ref(false)

  async function fetchGroups() {
    try {
      const res = await api.getGroups()
      if (res.code === 0) {
        groups.value = res.data
      }
    } catch (e) {
      console.error('Failed to fetch groups', e)
    }
  }

  async function fetchConnections() {
    loading.value = true
    try {
      await fetchGroups()
      const res = await api.getConnections()
      if (res.code === 0) {
        connections.value = res.data
      }
    } finally {
      loading.value = false
    }
  }

  async function createGroup(data: any) {
    const res = await api.createGroup(data)
    if (res.code === 0) {
      await fetchGroups()
      return res.data
    }
    throw new Error(res.message)
  }

  async function deleteGroup(id: string) {
    const res = await api.deleteGroup(id)
    if (res.code === 0) {
      await fetchGroups()
    } else {
      throw new Error(res.message)
    }
  }

  async function createConnection(data: any) {
    const res = await api.createConnection(data)
    if (res.code === 0) {
      await fetchConnections()
      return res.data
    }
    throw new Error(res.message)
  }

  async function updateConnection(id: string, data: any) {
    const res = await api.updateConnection(id, data)
    if (res.code === 0) {
      await fetchConnections()
      return res.data
    }
    throw new Error(res.message)
  }

  async function deleteConnection(id: string) {
    const res = await api.deleteConnection(id)
    if (res.code === 0) {
      await fetchConnections()
    }
  }

  async function testConnection(id: string) {
    const res = await api.testConnection(id)
    return res.code === 0 ? res.data : null
  }

  function getConnectionById(id: string) {
    return connections.value.find((c) => c.id === id)
  }

  return {
    connections,
    groups,
    loading,
    fetchGroups,
    fetchConnections,
    createGroup,
    deleteGroup,
    createConnection,
    updateConnection,
    deleteConnection,
    testConnection,
    getConnectionById
  }
})
