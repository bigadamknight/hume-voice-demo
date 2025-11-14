import { useState, useEffect } from 'react'
import { voices, Voice, CreateVoiceRequest, auth } from '../lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from './ui/table'

export function VoiceAdmin() {
  const [voiceList, setVoiceList] = useState<Voice[]>([])
  const [loading, setLoading] = useState(true)
  const [isAdmin, setIsAdmin] = useState<boolean | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState<CreateVoiceRequest>({
    name: '',
    description: '',
    prompt: '',
    voice_description: '',
    evi_version: '3',
    temperature: 1.0,
  })
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [syncing, setSyncing] = useState<string | null>(null) // voice ID being synced
  const [syncingAll, setSyncingAll] = useState(false)

  useEffect(() => {
    checkAdmin()
  }, [])

  useEffect(() => {
    if (isAdmin === true) {
      loadVoices()
    }
  }, [isAdmin])

  const checkAdmin = async () => {
    try {
      const user = await auth.me()
      setIsAdmin(user.is_admin)
    } catch {
      setIsAdmin(false)
    }
  }

  const loadVoices = async () => {
    try {
      setLoading(true)
      const data = await voices.list()
      setVoiceList(data || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load voices')
      setVoiceList([])
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      await voices.create(formData)
      setShowForm(false)
      setFormData({
        name: '',
        description: '',
        prompt: '',
        voice_description: '',
        evi_version: '3',
        temperature: 1.0,
      })
      await loadVoices()
    } catch (err: any) {
      const errorMessage = err?.response?.data || err?.message || 'Failed to create voice'
      setError(typeof errorMessage === 'string' ? errorMessage : JSON.stringify(errorMessage))
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this voice?')) {
      return
    }

    try {
      await voices.delete(id)
      await loadVoices()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete voice')
    }
  }

  const handleSync = async (id: string) => {
    setSyncing(id)
    setError(null)
    try {
      await voices.sync(id)
      alert('Voice synced to Hume successfully!')
    } catch (err: any) {
      const errorMessage = err?.response?.data || err?.message || 'Failed to sync voice'
      setError(typeof errorMessage === 'string' ? errorMessage : JSON.stringify(errorMessage))
    } finally {
      setSyncing(null)
    }
  }

  const handleSyncAll = async () => {
    if (!confirm('This will sync all voices to Hume. Continue?')) {
      return
    }
    setSyncingAll(true)
    setError(null)
    try {
      const result = await voices.syncAll()
      const synced = result.results.filter(r => r.status === 'synced').length
      const errors = result.results.filter(r => r.status === 'error').length
      const skipped = result.results.filter(r => r.status === 'skipped').length
      alert(`Sync completed:\n- Synced: ${synced}\n- Errors: ${errors}\n- Skipped: ${skipped}`)
      if (errors > 0) {
        const errorDetails = result.results
          .filter(r => r.status === 'error')
          .map(r => `${r.name}: ${r.error}`)
          .join('\n')
        setError(`Some voices failed to sync:\n${errorDetails}`)
      }
    } catch (err: any) {
      const errorMessage = err?.response?.data || err?.message || 'Failed to sync voices'
      setError(typeof errorMessage === 'string' ? errorMessage : JSON.stringify(errorMessage))
    } finally {
      setSyncingAll(false)
    }
  }

  if (isAdmin === null || loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="text-center">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent"></div>
          <p className="mt-4 text-sm text-muted-foreground">Loading...</p>
        </div>
      </div>
    )
  }

  if (!isAdmin) {
    return (
      <div className="container mx-auto px-4 py-8 max-w-6xl">
        <Card>
          <CardContent className="pt-6">
            <div className="text-center">
              <h2 className="text-2xl font-bold text-red-600 mb-2">Access Denied</h2>
              <p className="text-muted-foreground">You must be an administrator to access this page.</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8 max-w-6xl">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold">Voice Management</h1>
          <p className="text-muted-foreground mt-1">Create and manage EVI voice configurations</p>
        </div>
        <div className="flex gap-2">
          <Button 
            onClick={handleSyncAll} 
            disabled={syncingAll || voiceList.length === 0}
            variant="outline"
          >
            {syncingAll ? 'Syncing...' : 'Sync All to Hume'}
          </Button>
          <Button onClick={() => setShowForm(!showForm)}>
            {showForm ? 'Cancel' : 'Add New Voice'}
          </Button>
        </div>
      </div>

      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-md">
          <p className="text-sm text-red-800">{error}</p>
        </div>
      )}

      {showForm && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create New Voice</CardTitle>
            <CardDescription>
              Enter the voice details. The system will create a Hume voice and EVI configuration automatically.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <Label htmlFor="name">Name *</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                  placeholder="e.g., Friendly Assistant"
                  className="bg-white"
                />
              </div>

              <div>
                <Label htmlFor="description">Description</Label>
                <Input
                  id="description"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Brief description of this voice"
                  className="bg-white"
                />
              </div>

              <div>
                <Label htmlFor="prompt">System Prompt *</Label>
                <textarea
                  id="prompt"
                  value={formData.prompt}
                  onChange={(e) => setFormData({ ...formData, prompt: e.target.value })}
                  required
                  rows={6}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Enter the system prompt for this voice..."
                />
              </div>

              <div>
                <Label htmlFor="voice_description">Voice Description</Label>
                <textarea
                  id="voice_description"
                  value={formData.voice_description}
                  onChange={(e) => setFormData({ ...formData, voice_description: e.target.value })}
                  rows={3}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g., A warm, friendly voice with a slight British accent, speaking at a moderate pace"
                />
                <p className="text-xs text-muted-foreground mt-1">
                  Describe how you want the voice to sound. This will be used to generate a custom voice via Hume TTS.
                </p>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="evi_version">EVI Version</Label>
                  <Input
                    id="evi_version"
                    type="text"
                    value={formData.evi_version}
                    onChange={(e) => setFormData({ ...formData, evi_version: e.target.value })}
                    placeholder="3"
                    className="bg-white"
                  />
                </div>

                <div>
                  <Label htmlFor="temperature">Temperature</Label>
                  <Input
                    id="temperature"
                    type="number"
                    step="0.1"
                    min="0"
                    max="2"
                    value={formData.temperature}
                    onChange={(e) => setFormData({ ...formData, temperature: parseFloat(e.target.value) || 1.0 })}
                    className="bg-white"
                  />
                </div>
              </div>

              <div className="flex gap-2">
                <Button type="submit" disabled={submitting}>
                  {submitting ? 'Creating...' : 'Create Voice'}
                </Button>
                <Button type="button" variant="outline" onClick={() => setShowForm(false)}>
                  Cancel
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Existing Voices</CardTitle>
          <CardDescription>
            {voiceList?.length || 0} voice{(voiceList?.length || 0) !== 1 ? 's' : ''} configured
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!voiceList || voiceList.length === 0 ? (
            <p className="text-center text-muted-foreground py-8">No voices created yet. Add your first voice above.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Hume Config ID</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(voiceList || []).map((voice) => (
                  <TableRow key={voice.id}>
                    <TableCell className="font-medium">{voice.name}</TableCell>
                    <TableCell className="max-w-xs truncate">{voice.description || '-'}</TableCell>
                    <TableCell className="font-mono text-xs">{voice.hume_config_id || 'Not created'}</TableCell>
                    <TableCell>{new Date(voice.created_at).toLocaleDateString()}</TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleSync(voice.id)}
                          disabled={syncing === voice.id || !voice.hume_config_id || !voice.prompt}
                        >
                          {syncing === voice.id ? 'Syncing...' : 'Sync'}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            navigator.clipboard.writeText(voice.hume_config_id)
                            alert('Config ID copied to clipboard!')
                          }}
                          disabled={!voice.hume_config_id}
                        >
                          Copy Config ID
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleDelete(voice.id)}
                          className="text-red-600 hover:text-red-700"
                        >
                          Delete
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

