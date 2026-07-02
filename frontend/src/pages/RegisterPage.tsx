import React, { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { mutate } from 'swr';
import { useAuth } from '@/contexts/AuthContext';
import type { SetupStatusResponse } from '@/lib/api/services/setup';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { APIError } from '@/lib/api/types';
import { ShootingStars } from '@/components/ui/shooting-stars';
import { StarsBackground } from '@/components/ui/stars-background';
import { EncryptedText } from '@/components/ui/encrypted-text';

export const RegisterPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { register } = useAuth();

  // Check if this is first user setup (from SetupCheck redirect)
  const isFirstUser = location.state?.isFirstUser || false;

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [emailError, setEmailError] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [confirmPasswordError, setConfirmPasswordError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');

  // Password strength calculation
  const calculatePasswordStrength = (pwd: string): number => {
    let strength = 0;
    if (pwd.length >= 8) strength += 25;
    if (pwd.length >= 12) strength += 25;
    if (/[a-z]/.test(pwd) && /[A-Z]/.test(pwd)) strength += 25;
    if (/\d/.test(pwd)) strength += 12.5;
    if (/[!@#$%^&*(),.?":{}|<>]/.test(pwd)) strength += 12.5;
    return Math.min(100, strength);
  };

  const getPasswordStrengthLabel = (strength: number): string => {
    if (strength === 0) return '';
    if (strength < 40) return 'Weak';
    if (strength < 70) return 'Fair';
    if (strength < 90) return 'Good';
    return 'Strong';
  };

  const getPasswordStrengthColor = (strength: number): string => {
    if (strength === 0) return 'bg-gray-200';
    if (strength < 40) return 'bg-red-500';
    if (strength < 70) return 'bg-yellow-500';
    if (strength < 90) return 'bg-blue-500';
    return 'bg-green-500';
  };

  const passwordStrength = calculatePasswordStrength(password);
  const strengthLabel = getPasswordStrengthLabel(passwordStrength);
  const strengthColor = getPasswordStrengthColor(passwordStrength);

  // Email validation
  const validateEmail = (value: string): boolean => {
    if (!value) {
      setEmailError('Email is required');
      return false;
    }
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(value)) {
      setEmailError('Invalid email format');
      return false;
    }
    setEmailError('');
    return true;
  };

  // Password validation
  const validatePassword = (value: string): boolean => {
    if (!value) {
      setPasswordError('Password is required');
      return false;
    }
    if (value.length < 8) {
      setPasswordError('Password must be at least 8 characters');
      return false;
    }
    setPasswordError('');
    return true;
  };

  // Confirm password validation
  const validateConfirmPassword = (value: string): boolean => {
    if (!value) {
      setConfirmPasswordError('Please confirm your password');
      return false;
    }
    if (value !== password) {
      setConfirmPasswordError('Passwords do not match');
      return false;
    }
    setConfirmPasswordError('');
    return true;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccessMessage('');

    // Validate all fields
    const isEmailValid = validateEmail(email);
    const isPasswordValid = validatePassword(password);
    const isConfirmPasswordValid = validateConfirmPassword(confirmPassword);

    if (!isEmailValid || !isPasswordValid || !isConfirmPasswordValid) {
      return;
    }

    setIsLoading(true);

    try {
      await register({ email, password });
      // A user now exists, so the system no longer requires initial setup.
      // Refresh the cached setup status (SetupCheck reads '/api/setup/status'
      // with revalidateIfStale disabled) so it does not bounce the /login
      // navigation below back to /register.
      await mutate<SetupStatusResponse>(
        '/api/setup/status',
        (current) => ({
          requires_setup: false,
          user_count: (current?.user_count ?? 0) + 1,
        }),
        { revalidate: false },
      );
      setSuccessMessage('Account created successfully! Redirecting to login...');
      // Redirect to login after 2 seconds
      setTimeout(() => {
        navigate('/login');
      }, 2000);
    } catch (err) {
      if (err instanceof APIError) {
        if (err.statusCode === 409 || err.message.includes('already exists')) {
          setError('An account with this email already exists.');
        } else {
          setError(err.message);
        }
      } else {
        setError('An unexpected error occurred. Please try again.');
      }
      setIsLoading(false);
    }
  };

  return (
    <div className="relative flex items-center justify-center min-h-screen overflow-hidden">
      {/* Animated background */}
      <StarsBackground className="absolute inset-0 -z-10" />
      <ShootingStars className="absolute inset-0 -z-10" />

      <Card className="w-full max-w-md">
        <CardHeader className="space-y-1">
          <CardTitle className="text-2xl font-bold text-center">
            <EncryptedText
              text={isFirstUser ? 'Initial Setup' : 'Create an account'}
              revealDelayMs={40}
            />
          </CardTitle>
          <CardDescription className="text-center">
            <EncryptedText
              text={
                isFirstUser
                  ? 'Create the first administrator account to get started'
                  : 'Enter your details to create your Deeb account'
              }
              revealDelayMs={25}
            />
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {isFirstUser && (
              <div className="p-3 text-sm text-blue-600 bg-blue-50 dark:bg-blue-900/20 dark:text-blue-400 rounded-md">
                You will be granted <strong>OrganizationAdmin</strong> privileges as the first user.
              </div>
            )}

            {error && (
              <div className="p-3 text-sm text-red-600 bg-red-50 dark:bg-red-900/20 dark:text-red-400 rounded-md">
                {error}
              </div>
            )}

            {successMessage && (
              <div className="p-3 text-sm text-green-600 bg-green-50 dark:bg-green-900/20 dark:text-green-400 rounded-md">
                {successMessage}
              </div>
            )}

            <div className="space-y-2">
              <label htmlFor="email" className="text-sm font-medium">
                Email
              </label>
              <Input
                id="email"
                type="email"
                placeholder="you@example.com"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  if (emailError) validateEmail(e.target.value);
                }}
                onBlur={() => validateEmail(email)}
                disabled={isLoading}
                className={emailError ? 'border-red-500' : ''}
              />
              {emailError && <p className="text-xs text-red-600 dark:text-red-400">{emailError}</p>}
            </div>

            <div className="space-y-2">
              <label htmlFor="password" className="text-sm font-medium">
                Password
              </label>
              <Input
                id="password"
                type="password"
                placeholder="Create a strong password"
                value={password}
                onChange={(e) => {
                  setPassword(e.target.value);
                  if (passwordError) validatePassword(e.target.value);
                }}
                onBlur={() => validatePassword(password)}
                disabled={isLoading}
                className={passwordError ? 'border-red-500' : ''}
              />
              {passwordError && (
                <p className="text-xs text-red-600 dark:text-red-400">{passwordError}</p>
              )}

              {/* Password strength indicator */}
              {password && (
                <div className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-gray-600 dark:text-gray-400">Password strength:</span>
                    <span
                      className={`font-medium ${
                        passwordStrength < 40
                          ? 'text-red-600'
                          : passwordStrength < 70
                            ? 'text-yellow-600'
                            : passwordStrength < 90
                              ? 'text-blue-600'
                              : 'text-green-600'
                      }`}
                    >
                      {strengthLabel}
                    </span>
                  </div>
                  <div className="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                    <div
                      className={`h-full transition-all duration-300 ${strengthColor}`}
                      style={{ width: `${passwordStrength}%` }}
                    />
                  </div>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    Use 8+ characters with a mix of letters, numbers & symbols
                  </p>
                </div>
              )}
            </div>

            <div className="space-y-2">
              <label htmlFor="confirmPassword" className="text-sm font-medium">
                Confirm Password
              </label>
              <Input
                id="confirmPassword"
                type="password"
                placeholder="Confirm your password"
                value={confirmPassword}
                onChange={(e) => {
                  setConfirmPassword(e.target.value);
                  if (confirmPasswordError) validateConfirmPassword(e.target.value);
                }}
                onBlur={() => validateConfirmPassword(confirmPassword)}
                disabled={isLoading}
                className={confirmPasswordError ? 'border-red-500' : ''}
              />
              {confirmPasswordError && (
                <p className="text-xs text-red-600 dark:text-red-400">{confirmPasswordError}</p>
              )}
            </div>

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? 'Creating account...' : 'Create account'}
            </Button>
          </form>
        </CardContent>
        {!isFirstUser && (
          <CardFooter className="flex justify-center">
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Already have an account?{' '}
              <a href="/login" className="text-blue-600 dark:text-blue-400 hover:underline">
                Sign in
              </a>
            </p>
          </CardFooter>
        )}
      </Card>
    </div>
  );
};
