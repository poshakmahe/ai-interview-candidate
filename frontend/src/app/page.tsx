'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth, useIsAuthenticated } from '@/hooks/useAuth';
import Header from '@/components/layout/Header';
import Button from '@/components/ui/Button';
import { Shield, Lock, Share2, Zap } from 'lucide-react';
import Link from 'next/link';

export default function HomePage() {
  const router = useRouter();
  const { isLoading } = useAuth();
  const isAuthenticated = useIsAuthenticated();

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.push('/dashboard');
    }
  }, [isAuthenticated, isLoading, router]);

  const features = [
    {
      icon: Shield,
      title: 'Secure Storage',
      description: 'Your documents are protected with industry-standard encryption.',
    },
    {
      icon: Lock,
      title: 'Access Control',
      description: 'Fine-grained permissions to control who can view or edit your files.',
    },
    {
      icon: Share2,
      title: 'Easy Sharing',
      description: 'Share documents securely with colleagues and collaborators.',
    },
    {
      icon: Zap,
      title: 'Fast & Reliable',
      description: 'Quick uploads and downloads with high availability.',
    },
  ];

  return (
    <>
      <Header />
      <main>
        {/* Hero Section */}
        <section className="py-20 px-4 sm:px-6 lg:px-8">
          <div className="max-w-4xl mx-auto text-center">
            <h1 className="text-4xl sm:text-5xl font-bold text-gray-900 mb-6">
              Secure Document Vault
            </h1>
            <p className="text-xl text-gray-600 mb-8">
              Store, manage, and share your sensitive documents with confidence.
              Enterprise-grade security for your most important files.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link href="/register">
                <Button size="lg">Get Started Free</Button>
              </Link>
              <Link href="/login">
                <Button variant="secondary" size="lg">Sign In</Button>
              </Link>
            </div>
          </div>
        </section>

        {/* Features Section */}
        <section className="py-16 px-4 sm:px-6 lg:px-8 bg-white">
          <div className="max-w-6xl mx-auto">
            <h2 className="text-3xl font-bold text-gray-900 text-center mb-12">
              Why Choose SecureVault?
            </h2>
            <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8">
              {features.map((feature) => (
                <div key={feature.title} className="text-center">
                  <div className="inline-flex items-center justify-center w-12 h-12 rounded-lg bg-primary-100 text-primary-600 mb-4">
                    <feature.icon className="h-6 w-6" />
                  </div>
                  <h3 className="text-lg font-semibold text-gray-900 mb-2">
                    {feature.title}
                  </h3>
                  <p className="text-gray-600">{feature.description}</p>
                </div>
              ))}
            </div>
          </div>
        </section>
      </main>
    </>
  );
}
