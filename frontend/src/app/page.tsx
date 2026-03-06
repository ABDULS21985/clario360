import { redirect } from 'next/navigation';

// Root — redirect to dashboard (middleware handles auth check and redirects to /login if needed)
export default function Home() {
  redirect('/dashboard');
}
