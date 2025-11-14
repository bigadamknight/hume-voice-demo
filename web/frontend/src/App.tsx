import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, Link, useLocation, useNavigate } from 'react-router-dom'
import { Auth } from './components/Auth'
import { ConversationList } from './components/ConversationList'
import { VoiceChat } from './components/VoiceChat'
import { Transcript } from './components/Transcript'
import { VoiceAdmin } from './components/VoiceAdmin'
import { UserAdmin } from './components/UserAdmin'
import { auth, conversations, User } from './lib/api'
import { Button } from './components/ui/button'

function AppContent() {
  const [authenticated, setAuthenticated] = useState(false)
  const [loading, setLoading] = useState(true)
  const [user, setUser] = useState<User | null>(null)
  const [selectedConversationId, setSelectedConversationId] = useState<string>('')
  const location = useLocation()
  const navigate = useNavigate()

  useEffect(() => {
    checkAuth()
  }, [])

  // Redirect non-admin users away from admin pages
  useEffect(() => {
    if (authenticated && user && !user.is_admin) {
      if (location.pathname.startsWith('/admin')) {
        navigate('/', { replace: true })
      }
    }
  }, [authenticated, user, location.pathname, navigate])

  const checkAuth = async () => {
    try {
      const userData = await auth.me()
      setUser(userData)
      setAuthenticated(true)
    } catch {
      setAuthenticated(false)
    } finally {
      setLoading(false)
    }
  }

  const handleLogin = () => {
    checkAuth()
  }

  const handleLogout = async () => {
    try {
      await auth.logout()
      setAuthenticated(false)
      setUser(null)
      setSelectedConversationId('')
    } catch (error) {
      console.error('Logout failed:', error)
    }
  }

  const handleCreateConversation = async () => {
    try {
      const conv = await conversations.create()
      setSelectedConversationId(conv.id)
    } catch (error) {
      console.error('Failed to create conversation:', error)
    }
  }

  // Show loading state during initial auth check
  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent align-[-0.125em] motion-reduce:animate-[spin_1.5s_linear_infinite]"></div>
          <p className="mt-4 text-sm text-muted-foreground">Loading...</p>
        </div>
      </div>
    )
  }

  // Show auth component only if not authenticated and not loading
  if (!authenticated) {
    return <Auth onLogin={handleLogin} />
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b">
        <div className="container mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-6">
            <h1 className="text-2xl font-bold">VOICE DEMO</h1>
            <nav className="flex gap-4">
              <Link
                to="/"
                className={`text-sm font-medium ${
                  location.pathname === '/' ? 'text-primary' : 'text-muted-foreground hover:text-primary'
                }`}
              >
                Conversations
              </Link>
              {user?.is_admin && (
                <>
                  <Link
                    to="/admin/users"
                    className={`text-sm font-medium ${
                      location.pathname === '/admin/users' ? 'text-primary' : 'text-muted-foreground hover:text-primary'
                    }`}
                  >
                    User Admin
                  </Link>
                  <Link
                    to="/admin/voices"
                    className={`text-sm font-medium ${
                      location.pathname === '/admin/voices' ? 'text-primary' : 'text-muted-foreground hover:text-primary'
                    }`}
                  >
                    Voice Admin
                  </Link>
                </>
              )}
            </nav>
          </div>
          <div className="flex items-center gap-4">
            <span className="text-sm text-muted-foreground">{user?.username}</span>
            <Button variant="outline" onClick={handleLogout}>
              Logout
            </Button>
          </div>
        </div>
      </header>
      <Routes>
        <Route
          path="/"
          element={
            <main className="container mx-auto px-4 py-8">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div>
                  <ConversationList
                    onSelectConversation={setSelectedConversationId}
                    onCreateConversation={handleCreateConversation}
                    selectedId={selectedConversationId}
                  />
                </div>
                <div className="space-y-6">
                  <VoiceChat
                    conversationId={selectedConversationId}
                    onConversationCreated={setSelectedConversationId}
                  />
                  <Transcript conversationId={selectedConversationId} />
                </div>
              </div>
            </main>
          }
        />
        <Route path="/admin/users" element={<UserAdmin />} />
        <Route path="/admin/voices" element={<VoiceAdmin />} />
      </Routes>
    </div>
  )
}

function App() {
  return (
    <BrowserRouter>
      <AppContent />
    </BrowserRouter>
  )
}

export default App

