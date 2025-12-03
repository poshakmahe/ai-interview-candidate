import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test.beforeEach(async ({ page, baseURL }) => {
    // Clear any existing auth state
    await page.context().clearCookies();
    // Navigate first to have a valid origin for localStorage access
    await page.goto(baseURL || '/');
    await page.evaluate(() => localStorage.clear());
  });

  test('should show login page', async ({ page }) => {
    await page.goto('/login');

    await expect(page.getByRole('heading', { name: 'Sign in to your account' })).toBeVisible();
    await expect(page.getByLabel('Email Address')).toBeVisible();
    await expect(page.getByLabel('Password')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Sign In' })).toBeVisible();
  });

  test('should show registration page', async ({ page }) => {
    await page.goto('/register');

    await expect(page.getByRole('heading', { name: 'Create your account' })).toBeVisible();
    await expect(page.getByLabel('Full Name')).toBeVisible();
    await expect(page.getByLabel('Email Address')).toBeVisible();
    await expect(page.getByLabel('Password')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Create Account' })).toBeVisible();
  });

  test('should show validation errors for empty login form', async ({ page }) => {
    await page.goto('/login');

    await page.getByRole('button', { name: 'Sign In' }).click();

    await expect(page.getByText('Please enter a valid email address')).toBeVisible();
    await expect(page.getByText('Password is required')).toBeVisible();
  });

  test('should show validation errors for invalid email in registration', async ({ page }) => {
    await page.goto('/register');

    await page.getByLabel('Full Name').fill('John');
    await page.getByLabel('Email Address').fill('invalid-email');
    await page.getByLabel('Password').fill('short');
    await page.getByRole('button', { name: 'Create Account' }).click();

    await expect(page.getByText('Please enter a valid email address')).toBeVisible();
    await expect(page.getByText('Password must be at least 8 characters')).toBeVisible();
  });

  test('should navigate between login and register pages', async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('link', { name: 'Sign up' }).click();

    await expect(page).toHaveURL('/register');

    await page.getByRole('link', { name: 'Sign in' }).click();

    await expect(page).toHaveURL('/login');
  });

  test('should redirect unauthenticated users from dashboard to login', async ({ page }) => {
    await page.goto('/dashboard');

    // Should redirect to login
    await expect(page).toHaveURL(/\/login/);
  });
});

test.describe('Home Page', () => {
  test('should show landing page for unauthenticated users', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByRole('heading', { name: 'Secure Document Vault' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Get Started Free' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Sign In' })).toBeVisible();
  });

  test('should show features section', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByText('Secure Storage')).toBeVisible();
    await expect(page.getByText('Access Control')).toBeVisible();
    await expect(page.getByText('Easy Sharing')).toBeVisible();
    await expect(page.getByText('Fast & Reliable')).toBeVisible();
  });
});
