import { useState, useEffect } from 'react'
import { users, AdminUser, CreateUserRequest, UpdateUserRequest } from '../lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from './ui/table'

export function UserAdmin() {
  const [userList, setUserList] = useState<AdminUser[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState<CreateUserRequest>({
    username: '',
    password: '',
    name: '',
    is_admin: false,
  })
  const [editingUserId, setEditingUserId] = useState<string | null>(null)
  const [editData, setEditData] = useState<UpdateUserRequest>({})
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadUsers()
  }, [])

  const loadUsers = async () => {
    try {
      setLoading(true)
      const data = await users.list()
      setUserList(data || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users')
      setUserList([])
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      // Send name as undefined if empty
      const createData: CreateUserRequest = {
        ...formData,
        name: formData.name?.trim() || undefined,
      }
      await users.create(createData)
      setShowForm(false)
      setFormData({
        username: '',
        password: '',
        name: '',
        is_admin: false,
      })
      await loadUsers()
    } catch (err: any) {
      const errorMessage = err?.response?.data || err?.message || 'Failed to create user'
      setError(typeof errorMessage === 'string' ? errorMessage : JSON.stringify(errorMessage))
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this user?')) {
      return
    }

    try {
      await users.delete(id)
      await loadUsers()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete user')
    }
  }

  const handleEdit = (user: AdminUser) => {
    setEditingUserId(user.id)
    setEditData({
      password: '',
      name: user.name || '',
      is_admin: user.is_admin,
    })
  }

  const handleUpdate = async (id: string) => {
    setSubmitting(true)
    setError(null)

    try {
      const updateData: UpdateUserRequest = {}
      if (editData.password && editData.password.trim() !== '') {
        updateData.password = editData.password
      }
      if (editData.name !== undefined) {
        updateData.name = editData.name.trim() || undefined
      }
      if (editData.is_admin !== undefined) {
        updateData.is_admin = editData.is_admin
      }

      await users.update(id, updateData)
      setEditingUserId(null)
      setEditData({})
      await loadUsers()
    } catch (err: any) {
      const errorMessage = err?.response?.data || err?.message || 'Failed to update user'
      setError(typeof errorMessage === 'string' ? errorMessage : JSON.stringify(errorMessage))
    } finally {
      setSubmitting(false)
    }
  }

  const cancelEdit = () => {
    setEditingUserId(null)
    setEditData({})
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="text-center">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent"></div>
          <p className="mt-4 text-sm text-muted-foreground">Loading users...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8 max-w-6xl">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold">User Management</h1>
          <p className="text-muted-foreground mt-1">Create and manage user accounts</p>
        </div>
        <Button onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Cancel' : 'Add New User'}
        </Button>
      </div>

      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-md">
          <p className="text-sm text-red-800">{error}</p>
        </div>
      )}

      {showForm && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create New User</CardTitle>
            <CardDescription>
              Enter the user details to create a new account.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <Label htmlFor="username">Username *</Label>
                <Input
                  id="username"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  required
                  placeholder="Enter username"
                  className="bg-white"
                />
              </div>

              <div>
                <Label htmlFor="password">Password *</Label>
                <Input
                  id="password"
                  type="password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  required
                  placeholder="Enter password"
                  className="bg-white"
                />
              </div>

              <div>
                <Label htmlFor="name">First Name</Label>
                <Input
                  id="name"
                  value={formData.name || ''}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter first name (optional)"
                  className="bg-white"
                />
                <p className="text-xs text-muted-foreground mt-1">
                  This name will be available as <code className="bg-gray-100 px-1 rounded">{'{{name}}'}</code> in voice prompts
                </p>
              </div>

              <div className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  id="is_admin"
                  checked={formData.is_admin}
                  onChange={(e) => setFormData({ ...formData, is_admin: e.target.checked })}
                  className="h-4 w-4"
                />
                <Label htmlFor="is_admin">Admin user</Label>
              </div>

              <div className="flex gap-2">
                <Button type="submit" disabled={submitting}>
                  {submitting ? 'Creating...' : 'Create User'}
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
          <CardTitle>Existing Users</CardTitle>
          <CardDescription>
            {userList?.length || 0} user{(userList?.length || 0) !== 1 ? 's' : ''} configured
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!userList || userList.length === 0 ? (
            <p className="text-center text-muted-foreground py-8">No users created yet. Add your first user above.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Username</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Admin</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(userList || []).map((user) => (
                  <TableRow key={user.id}>
                    <TableCell className="font-medium">{user.username}</TableCell>
                    <TableCell>{user.name || '-'}</TableCell>
                    <TableCell>{user.is_admin ? 'Yes' : 'No'}</TableCell>
                    <TableCell>{new Date(user.created_at).toLocaleDateString()}</TableCell>
                    <TableCell>
                      {editingUserId === user.id ? (
                        <div className="flex gap-2 items-center">
                          <Input
                            type="password"
                            placeholder="New password (leave empty to keep)"
                            value={editData.password || ''}
                            onChange={(e) => setEditData({ ...editData, password: e.target.value })}
                            className="w-48 bg-white"
                          />
                          <Input
                            type="text"
                            placeholder="First name"
                            value={editData.name || ''}
                            onChange={(e) => setEditData({ ...editData, name: e.target.value })}
                            className="w-32 bg-white"
                          />
                          <label className="flex items-center space-x-2">
                            <input
                              type="checkbox"
                              checked={editData.is_admin ?? user.is_admin}
                              onChange={(e) => setEditData({ ...editData, is_admin: e.target.checked })}
                              className="h-4 w-4"
                            />
                            <span className="text-sm">Admin</span>
                          </label>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleUpdate(user.id)}
                            disabled={submitting}
                          >
                            Save
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={cancelEdit}
                            disabled={submitting}
                          >
                            Cancel
                          </Button>
                        </div>
                      ) : (
                        <div className="flex gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleEdit(user)}
                          >
                            Edit
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleDelete(user.id)}
                            className="text-red-600 hover:text-red-700"
                          >
                            Delete
                          </Button>
                        </div>
                      )}
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

