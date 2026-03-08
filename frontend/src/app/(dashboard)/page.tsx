import { redirect } from 'next/navigation';

// Root within dashboard layout group — redirect to /dashboard
export default function DashboardRoot() {
  redirect('/dashboard');
}
