import { useState, useEffect } from 'react';
import { healthService, type HealthCheckResponse, APIError } from '../lib/api';
import { Card } from './ui/card';
import { Button } from './ui/button';

export function HealthCheck() {
  const [health, setHealth] = useState<HealthCheckResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const checkHealth = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await healthService.check();
      setHealth(response);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('An unexpected error occurred');
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // Check health on mount
    checkHealth();
  }, []);

  return (
    <Card className="p-6 max-w-md">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold">API Health Check</h2>
          <Button onClick={checkHealth} disabled={loading} variant="outline" size="sm">
            {loading ? 'Checking...' : 'Refresh'}
          </Button>
        </div>

        {error && (
          <div className="p-4 rounded-md bg-red-50 border border-red-200">
            <p className="text-sm text-red-800">
              <strong>Error:</strong> {error}
            </p>
          </div>
        )}

        {health && (
          <div className="space-y-3">
            <div className="flex items-center justify-between p-3 rounded-md bg-green-50 border border-green-200">
              <span className="font-medium text-green-900">Status</span>
              <span className="text-green-700">{health.status}</span>
            </div>

            <div className="flex items-center justify-between p-3 rounded-md bg-gray-50 border border-gray-200">
              <span className="font-medium text-gray-900">Version</span>
              <span className="text-gray-700">{health.version}</span>
            </div>

            <div className="flex items-center justify-between p-3 rounded-md bg-gray-50 border border-gray-200">
              <span className="font-medium text-gray-900">Timestamp</span>
              <span className="text-gray-700 text-sm">
                {new Date(health.timestamp).toLocaleString()}
              </span>
            </div>

            <div className="flex items-center justify-between p-3 rounded-md bg-blue-50 border border-blue-200">
              <span className="font-medium text-blue-900">Database</span>
              <span className="text-blue-700">{health.database}</span>
            </div>
          </div>
        )}

        {!health && !error && !loading && (
          <div className="text-center text-gray-500 py-8">Click refresh to check API health</div>
        )}
      </div>
    </Card>
  );
}
