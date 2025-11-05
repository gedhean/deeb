import { useState } from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { ModeToggle } from '@/components/mode-toggle';
import { Link } from 'react-router-dom';
import { LoadingSpinner } from '@/components/ui/loading-spinner';

export function Home() {
  const { isAuthenticated, isLoading } = useAuth();
  const [count, setCount] = useState(0);
  const [name, setName] = useState('');

  // Show loading spinner while checking authentication
  if (isLoading) {
    return <LoadingSpinner fullScreen text="Loading..." />;
  }

  // Redirect authenticated users to dashboard
  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  return (
    <div className="min-h-screen bg-background p-8">
      <div className="max-w-4xl mx-auto space-y-8">
        <div className="flex justify-end">
          <ModeToggle />
        </div>
        <header className="text-center space-y-2">
          <h1 className="text-4xl font-bold text-foreground">Deeb</h1>
          <p className="text-xl text-muted-foreground">Database Change Management System</p>
        </header>

        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Welcome to Deeb MVP</CardTitle>
              <CardDescription>
                Secure database change workflow without direct database access
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Deeb provides an issue-based workflow for managing database changes with:
              </p>
              <ul className="list-disc list-inside text-sm space-y-1 text-muted-foreground">
                <li>Multi-approval processes</li>
                <li>Automated SQL validation</li>
                <li>Comprehensive audit logging</li>
                <li>Environment promotion (Staging → Production)</li>
              </ul>
            </CardContent>
            <CardFooter>
              <p className="text-xs text-muted-foreground">Frontend setup complete ✅</p>
            </CardFooter>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Component Demo</CardTitle>
              <CardDescription>Shadcn/ui components with Tailwind CSS</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="name" className="text-sm font-medium">
                  Enter your name
                </label>
                <Input
                  id="name"
                  type="text"
                  placeholder="John Doe"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
                {name && <p className="text-sm text-muted-foreground">Hello, {name}!</p>}
              </div>

              <div className="space-y-2">
                <p className="text-sm font-medium">Button counter</p>
                <div className="flex gap-2">
                  <Button onClick={() => setCount(count + 1)}>Count: {count}</Button>
                  <Button variant="outline" onClick={() => setCount(0)}>
                    Reset
                  </Button>
                </div>
              </div>
            </CardContent>
            <CardFooter className="flex gap-2">
              <Button variant="secondary" size="sm">
                Secondary
              </Button>
              <Button variant="destructive" size="sm">
                Destructive
              </Button>
              <Button variant="ghost" size="sm">
                Ghost
              </Button>
            </CardFooter>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>API Client Setup</CardTitle>
            <CardDescription>Test the API client integration</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground mb-4">
              The API client has been configured with JWT token management, request/response
              interceptors, and TypeScript type definitions. Visit the health check page to test it.
            </p>
            <Link to="/health">
              <Button>View Health Check</Button>
            </Link>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
