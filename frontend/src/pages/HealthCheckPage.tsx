import { HealthCheck } from '@/components/HealthCheck';
import { ModeToggle } from '@/components/mode-toggle';
import { Button } from '@/components/ui/button';
import { Link } from 'react-router-dom';

export function HealthCheckPage() {
  return (
    <div className="min-h-screen bg-background p-8">
      <div className="max-w-4xl mx-auto space-y-8">
        <div className="flex justify-between items-center">
          <Link to="/">
            <Button variant="ghost">← Back to Home</Button>
          </Link>
          <ModeToggle />
        </div>

        <div className="flex justify-center">
          <HealthCheck />
        </div>
      </div>
    </div>
  );
}
