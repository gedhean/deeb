import { useState } from 'react'
import './App.css'

function App() {
  const [count, setCount] = useState(0)

  return (
    <div className="app">
      <header className="app-header">
        <h1>Deeb - Database Change Management</h1>
        <p>Secure database change workflow without direct database access</p>
      </header>

      <main className="app-main">
        <div className="card">
          <button onClick={() => setCount((count) => count + 1)}>
            Count is {count}
          </button>
          <p>
            Edit <code>src/App.tsx</code> and save to test hot module replacement
          </p>
        </div>
      </main>

      <footer className="app-footer">
        <p>Frontend setup complete ✅</p>
      </footer>
    </div>
  )
}

export default App
