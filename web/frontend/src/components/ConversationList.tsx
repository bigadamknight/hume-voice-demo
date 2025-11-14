import { useEffect, useState } from 'react'
import { Trash2 } from 'lucide-react'
import { Button } from './ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from './ui/table'
import { conversations, Conversation } from '@/lib/api'

interface ConversationListProps {
  onSelectConversation: (id: string) => void
  onCreateConversation: () => void
  selectedId?: string
}

export function ConversationList({ onSelectConversation, onCreateConversation, selectedId }: ConversationListProps) {
  const [convs, setConvs] = useState<Conversation[]>([])
  const [loading, setLoading] = useState(true)

  const loadConversations = async () => {
    try {
      const data = await conversations.list()
      setConvs(Array.isArray(data) ? data : [])
    } catch (error) {
      console.error('Failed to load conversations:', error)
      setConvs([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadConversations()
  }, [])

  const handleDelete = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    if (confirm('Are you sure you want to delete this conversation?')) {
      try {
        await conversations.delete(id)
        await loadConversations()
        if (selectedId === id) {
          onSelectConversation('')
        }
      } catch (error) {
        console.error('Failed to delete conversation:', error)
      }
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Conversations</CardTitle>
            <CardDescription>Your voice conversation history</CardDescription>
          </div>
          <Button onClick={onCreateConversation}>New</Button>
        </div>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        {loading ? (
          <div className="text-center py-8 text-muted-foreground">Loading...</div>
        ) : convs.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            No conversations yet. Start a new one!
          </div>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="min-w-[120px]">Title</TableHead>
                  <TableHead className="w-20">Status</TableHead>
                  <TableHead className="w-16 text-center">Msgs</TableHead>
                  <TableHead className="w-24">Updated</TableHead>
                  <TableHead className="w-12"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {convs.map((conv) => (
                  <TableRow
                    key={conv.id}
                    className={selectedId === conv.id ? 'bg-muted' : 'cursor-pointer'}
                    onClick={() => onSelectConversation(conv.id)}
                  >
                    <TableCell className="font-medium truncate max-w-[120px]" title={conv.title || 'Untitled'}>
                      {conv.title || 'Untitled'}
                    </TableCell>
                    <TableCell>
                      <span className={`px-2 py-1 rounded text-xs whitespace-nowrap ${
                        conv.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {conv.status}
                      </span>
                    </TableCell>
                    <TableCell className="text-center">{conv.message_count || 0}</TableCell>
                    <TableCell className="text-xs whitespace-nowrap">
                      {new Date(conv.updated_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
                    </TableCell>
                    <TableCell className="w-12">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => handleDelete(conv.id, e)}
                        className="text-red-600 hover:text-red-700 hover:bg-red-50 p-1 h-8 w-8"
                        title="Delete conversation"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

