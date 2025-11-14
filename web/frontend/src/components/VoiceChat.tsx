import { useEffect, useRef, useState } from 'react'
import { Button } from './ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { Select } from './ui/select'
import { Label } from './ui/label'
import { voices, Voice } from '../lib/api'
import { 
  HumeClient,
  EVIWebAudioPlayer,
  getAudioStream,
  convertBlobToBase64,
  getBrowserSupportedMimeType,
  ensureSingleValidAudioTrack,
  MimeType,
} from 'hume'

interface VoiceChatProps {
  conversationId?: string
  onConversationCreated?: (id: string) => void
}

// Get Hume credentials from environment or use defaults for local dev
const HUME_API_KEY = import.meta.env.VITE_HUME_API_KEY || 'YXGl4tqjGOMgJwkZXuokXLOLhGGVoMAKCtAGhhGYXRs2wfbZ'
const HUME_SECRET_KEY = import.meta.env.VITE_HUME_SECRET_KEY || 'WgrPSAgAoALHDiCGbwynVnwOdiCGwSt2sDQNm8PWUCSxXUblSUm0AjhEOzsna4DM'
const DEFAULT_CONFIG_ID = import.meta.env.VITE_HUME_CONFIG_ID || '148c9f9b-f8c8-44d1-8320-4e9ef0f0cea5'

export function VoiceChat({ conversationId, onConversationCreated }: VoiceChatProps) {
  const [connected, setConnected] = useState(false)
  const [status, setStatus] = useState<'idle' | 'listening' | 'speaking' | 'processing'>('idle')
  const [voiceList, setVoiceList] = useState<Voice[]>([])
  const [selectedVoiceId, setSelectedVoiceId] = useState<string>('')
  const [loadingVoices, setLoadingVoices] = useState(true)
  const socketRef = useRef<any>(null)
  const recorderRef = useRef<MediaRecorder | null>(null)
  const playerRef = useRef<EVIWebAudioPlayer | null>(null)
  const [currentConversationId, setCurrentConversationId] = useState<string | undefined>(conversationId)
  const chatGroupIdRef = useRef<string | undefined>(undefined)
  const reconnectCountRef = useRef<number>(0)
  const conversationHistoryRef = useRef<Array<{role: string, content: string}>>([])

  useEffect(() => {
    loadVoices()
    return () => {
      disconnect()
    }
  }, [])

  const loadVoices = async () => {
    try {
      setLoadingVoices(true)
      const data = await voices.list()
      setVoiceList(data || [])
      // Auto-select first voice if available
      if (data && data.length > 0 && data[0].hume_config_id) {
        setSelectedVoiceId(data[0].id)
      }
    } catch (err) {
      console.error('Failed to load voices:', err)
    } finally {
      setLoadingVoices(false)
    }
  }

  const getSelectedConfigId = (): string => {
    if (!selectedVoiceId) {
      return DEFAULT_CONFIG_ID
    }
    const selectedVoice = voiceList.find(v => v.id === selectedVoiceId)
    return selectedVoice?.hume_config_id || DEFAULT_CONFIG_ID
  }

  const connect = async () => {
    try {
      setStatus('processing') // Show loading state immediately
      console.log('🔑 Starting connection process...')
      
      // Auto-create conversation if none selected
      let convId = conversationId || currentConversationId
      if (!convId) {
        console.log('📝 Creating new conversation...')
        const conv = await fetch('/api/conversations', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({}),
        })
        const convData = await conv.json()
        convId = convData.id
        setCurrentConversationId(convId)
        if (onConversationCreated && convId) {
          onConversationCreated(convId)
        }
        console.log('✅ Conversation created:', convId)
      }
      
      const configId = getSelectedConfigId()
      
      console.log('API Key:', HUME_API_KEY.substring(0, 10) + '...')
      console.log('Secret Key:', HUME_SECRET_KEY.substring(0, 10) + '...')
      console.log('Config ID:', configId)
      
      console.log('🔗 Connecting to EVI with config:', configId)
      
      // Initialize Hume client with apiKey (as per docs)
      const client = new HumeClient({
        apiKey: HUME_API_KEY,
      })
      
      console.log('✅ HumeClient initialized')

      // Connect to EVI - pass apiKey to connect() method to add it to WebSocket URL
      // If we have a chat_group_id from a previous session, resume it
      const connectOptions: any = {
        configId: configId,
        apiKey: HUME_API_KEY,
      }
      
      if (chatGroupIdRef.current) {
        connectOptions.resumedChatGroupId = chatGroupIdRef.current
        console.log('🔄 Resuming chat group:', chatGroupIdRef.current)
        reconnectCountRef.current += 1
        alert(`Connection lost. Reconnecting... (attempt ${reconnectCountRef.current})\nConversation context will be preserved.`)
      }
      
      const socket = await client.empathicVoice.chat.connect(connectOptions)
      console.log('✅ Socket created, waiting for connection...')

      socketRef.current = socket

      // Initialize audio player (as per docs)
      const player = new EVIWebAudioPlayer()
      playerRef.current = player

      // Handle socket events (following docs pattern)
      socket.on('open', async () => {
        console.log('✅ Socket opened - Hume EVI connected')
        setConnected(true)
        setStatus('listening')
        
        // Initialize player (as per docs)
        await player.init()
        console.log('✅ Audio player initialized')
        
        // Start audio capture using SDK helpers (as per docs)
        try {
          const mimeTypeResult = getBrowserSupportedMimeType()
          const mimeType = mimeTypeResult.success 
            ? mimeTypeResult.mimeType 
            : MimeType.WEBM
          
          const micAudioStream = await getAudioStream()
          ensureSingleValidAudioTrack(micAudioStream)
          
          const recorder = new MediaRecorder(micAudioStream, { mimeType })
          
          recorder.ondataavailable = async (e: BlobEvent) => {
            if (e.data.size > 0 && socket.readyState === WebSocket.OPEN) {
              const data = await convertBlobToBase64(e.data)
              socket.sendAudioInput({ data })
            }
          }
          
          recorder.onerror = (e) => console.error('MediaRecorder error:', e)
          recorder.start(80) // 80ms as per docs
          
          recorderRef.current = recorder
          console.log('✅ Audio capture started')
        } catch (error) {
          console.error('Failed to start audio capture:', error)
          alert('Failed to access microphone. Please check permissions.')
        }
      })

      socket.on('message', async (message: any) => {
        console.log('📨 Hume message:', message.type)

        switch (message.type) {
          case 'chat_metadata':
            // Store the chat_group_id for resuming on reconnect
            if (message.chat_group_id) {
              chatGroupIdRef.current = message.chat_group_id
              console.log('💾 Stored chat_group_id:', message.chat_group_id)
            }
            break
            
          case 'user_message':
            setStatus('processing')
            console.log('👤 User:', message.message.content)
            
            // Store in conversation history for context injection
            conversationHistoryRef.current.push({
              role: 'user',
              content: message.message.content
            })
            
            if (convId) {
              await saveMessage(convId, 'user', message.message.content)
            }
            
            // Example: Monitor conversation and inject context
            // You can call an external AI API here to analyze the conversation
            // and inject context to guide EVI's response
            // TODO: Uncomment when ready to enable AI-guided context injection
            // await monitorAndInjectContext(socket, conversationHistoryRef.current)
            break

          case 'assistant_message':
            setStatus('speaking')
            console.log('🤖 Assistant:', message.message.content)
            
            // Store in conversation history
            conversationHistoryRef.current.push({
              role: 'assistant',
              content: message.message.content
            })
            
            if (convId) {
              await saveMessage(convId, 'assistant', message.message.content)
            }
            break

          case 'audio_output':
            // Enqueue audio for playback (as per docs)
            await player.enqueue(message)
            setStatus('speaking')
            break

          case 'user_interruption':
            // Stop playback on interruption (as per docs)
            player.stop()
            setStatus('listening')
            break

          case 'error':
            console.error('Hume error:', message)
            alert(`Error: ${message.message}`)
            break
        }
      })

      socket.on('error', (error: Event | Error) => {
        console.error('Socket error:', error)
        setStatus('idle')
      })

      socket.on('close', (e: any) => {
        console.log('Socket closed:', e)
        setConnected(false)
        setStatus('idle')
        
        // Cleanup audio capture (as per docs)
        if (recorderRef.current) {
          recorderRef.current.stream.getTracks().forEach(track => track.stop())
          recorderRef.current = null
        }
        
        // Dispose player (as per docs)
        if (playerRef.current) {
          playerRef.current.dispose()
          playerRef.current = null
        }
      })

    } catch (error) {
      console.error('Failed to connect to Hume:', error)
      alert('Failed to start voice chat')
      setStatus('idle')
    }
  }

  const disconnect = () => {
    if (socketRef.current) {
      socketRef.current.close()
      socketRef.current = null
    }
    if (recorderRef.current) {
      recorderRef.current.stream.getTracks().forEach(track => track.stop())
      recorderRef.current = null
    }
    if (playerRef.current) {
      playerRef.current.dispose()
      playerRef.current = null
    }
    // Clear chat group ID on manual disconnect
    chatGroupIdRef.current = undefined
    reconnectCountRef.current = 0
    setConnected(false)
    setStatus('idle')
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Voice Chat</CardTitle>
        <CardDescription>Start a conversation with EVI</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {!connected && (
          <div className="space-y-2">
            <Label htmlFor="voice-select">Select Voice</Label>
            <Select
              id="voice-select"
              value={selectedVoiceId}
              onChange={(e) => setSelectedVoiceId(e.target.value)}
              disabled={loadingVoices || connected}
            >
              {loadingVoices ? (
                <option>Loading voices...</option>
              ) : voiceList.length === 0 ? (
                <option value="">No voices available</option>
              ) : (
                <>
                  {voiceList.map((voice) => (
                    <option key={voice.id} value={voice.id} disabled={!voice.hume_config_id}>
                      {voice.name} {!voice.hume_config_id ? '(No config)' : ''}
                    </option>
                  ))}
                </>
              )}
            </Select>
            {selectedVoiceId && (
              <p className="text-xs text-muted-foreground">
                {voiceList.find(v => v.id === selectedVoiceId)?.description || ''}
              </p>
            )}
          </div>
        )}
        <div className="flex items-center justify-center h-32">
          <div className={`w-24 h-24 rounded-full flex items-center justify-center ${
            status === 'listening' ? 'bg-green-100 animate-pulse' :
            status === 'speaking' ? 'bg-blue-100 animate-pulse' :
            status === 'processing' ? 'bg-yellow-100 animate-pulse' :
            'bg-gray-100'
          }`}>
            <span className="text-2xl">
              {status === 'listening' ? '🎤' :
               status === 'speaking' ? '🔊' :
               status === 'processing' ? '⏳' :
               '💬'}
            </span>
          </div>
        </div>
        <div className="text-center">
          <p className="text-sm text-muted-foreground capitalize">{status}</p>
          {connected && (
            <p className="text-xs text-muted-foreground mt-2">
              Echo cancellation enabled • 30-min session limit
            </p>
          )}
        </div>
        <div className="flex justify-center gap-2">
          {!connected ? (
            <Button 
              onClick={connect} 
              size="lg"
              disabled={loadingVoices || !selectedVoiceId || !voiceList.find(v => v.id === selectedVoiceId)?.hume_config_id}
            >
              Start Conversation
            </Button>
          ) : (
            <Button onClick={disconnect} variant="destructive" size="lg">
              End Conversation
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

async function saveMessage(conversationId: string, role: string, content: string) {
  try {
    await fetch(`/api/conversations/${conversationId}/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ role, content }),
    })
  } catch (error) {
    console.error('Failed to save message:', error)
  }
}

/**
 * Monitor conversation and inject context to guide EVI's responses.
 * This is called after each user message.
 * 
 * You can integrate an external AI model here to:
 * - Analyze conversation sentiment/topics
 * - Detect when to intervene (e.g., user is stressed, needs redirection)
 * - Inject context to guide EVI's tone or focus
 * 
 * @param socket - The EVI WebSocket connection
 * @param history - Array of conversation messages
 */
// @ts-expect-error - Function reserved for future use
async function monitorAndInjectContext(
  socket: any,
  history: Array<{role: string, content: string}>
) {
  try {
    // Example: Only monitor after a few messages to have enough context
    if (history.length < 3) return
    
    // Call backend API to analyze conversation with external AI
    const analysis = await fetch('/api/analyze-conversation', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ history }),
    })
    
    if (analysis.ok) {
      const { shouldIntervene, contextText, contextType, reasoning } = await analysis.json()
      
      if (shouldIntervene && contextText) {
        console.log('🧠 AI Analysis:', reasoning)
        injectContext(socket, contextText, contextType || 'temporary')
      }
    }
    
    // Example: Simple rule-based context injection
    const lastUserMessage = history[history.length - 1]
    if (lastUserMessage.role === 'user') {
      const content = lastUserMessage.content.toLowerCase()
      
      // Example: Detect stress-related keywords
      if (content.includes('stress') || content.includes('overwhelm') || content.includes('anxious')) {
        console.log('🧠 Detected stress - injecting supportive context')
        injectContext(socket, 
          'The user seems stressed. Be extra empathetic and supportive. Offer to help break down problems into smaller steps.',
          'temporary'
        )
      }
      
      // Example: Detect work-related topics
      if (content.includes('work') || content.includes('project') || content.includes('deadline')) {
        console.log('🧠 Detected work topic - injecting productivity context')
        injectContext(socket,
          'The user is discussing work. Focus on productivity, help them prioritize, and offer actionable suggestions.',
          'persistent'
        )
      }
    }
  } catch (error) {
    console.error('Failed to monitor/inject context:', error)
  }
}

/**
 * Inject context into the EVI session to guide responses.
 * See: https://dev.hume.ai/docs/speech-to-speech-evi/features/context-injection
 * 
 * @param socket - The EVI WebSocket connection
 * @param text - Context text to inject
 * @param type - 'temporary' (one response) or 'persistent' (all future responses)
 */
function injectContext(socket: any, text: string, type: 'temporary' | 'persistent') {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    console.warn('Cannot inject context: socket not open')
    return
  }
  
  const sessionSettings = {
    type: 'session_settings',
    context: {
      text,
      type
    }
  }
  
  console.log('💉 Injecting context:', type, '-', text.substring(0, 50) + '...')
  socket.sendSessionSettings(sessionSettings)
}

/**
 * Clear all injected context from the session.
 * 
 * @param socket - The EVI WebSocket connection
 */
function clearContext(socket: any) {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    console.warn('Cannot clear context: socket not open')
    return
  }
  
  const sessionSettings = {
    type: 'session_settings',
    context: null
  }
  
  console.log('🧹 Clearing injected context')
  socket.sendSessionSettings(sessionSettings)
}

// Export for use in other components if needed
export { injectContext, clearContext }
