import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { messages, Message } from '@/lib/api'

interface TranscriptProps {
  conversationId?: string
}

export function Transcript({ conversationId }: TranscriptProps) {
  const [msgs, setMsgs] = useState<Message[]>([])
  const [loading, setLoading] = useState(true)

  const loadMessages = async () => {
    if (!conversationId) {
      setMsgs([])
      setLoading(false)
      return
    }
    try {
      const data = await messages.list(conversationId)
      setMsgs(Array.isArray(data) ? data : [])
    } catch (error) {
      console.error('Failed to load messages:', error)
      setMsgs([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (conversationId) {
      loadMessages()
      // Poll for new messages
      const interval = setInterval(loadMessages, 2000)
      return () => clearInterval(interval)
    }
  }, [conversationId])

  if (!conversationId) {
    return null
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Transcript</CardTitle>
        <CardDescription>Conversation history</CardDescription>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="text-center py-8 text-muted-foreground">Loading...</div>
        ) : msgs.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">No messages yet</div>
        ) : (
          <div className="space-y-4 max-h-96 overflow-y-auto">
            {msgs.map((msg) => (
              <div
                key={msg.id}
                className={`p-3 rounded-lg ${
                  msg.role === 'user' ? 'bg-blue-50 ml-8' : 'bg-gray-50 mr-8'
                }`}
              >
                <div className="text-xs text-muted-foreground mb-1">
                  {msg.role === 'user' ? 'You' : 'EVI'} • {new Date(msg.timestamp).toLocaleTimeString()}
                </div>
                <div className="text-sm">{msg.content}</div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

