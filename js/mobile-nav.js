/**
 * Mobile Navigation
 * Handles hamburger menu toggle, keyboard interactions, and accessibility
 */

(function() {
  'use strict';

  const navToggle = document.querySelector('.nav-toggle');
  const navMenu = document.querySelector('.nav-menu');
  const navBackdrop = document.querySelector('.nav-menu-backdrop');

  if (!navToggle || !navMenu) {
    console.warn('Mobile navigation elements not found');
    return;
  }

  // Create backdrop element if it doesn't exist
  if (!navBackdrop) {
    const backdrop = document.createElement('div');
    backdrop.className = 'nav-menu-backdrop';
    document.body.appendChild(backdrop);
  }

  const backdrop = navBackdrop || document.querySelector('.nav-menu-backdrop');

  /**
   * Toggle mobile menu open/closed
   * @param {boolean} [forceState] - Force specific state (true=open, false=close)
   */
  function toggleMenu(forceState) {
    const isOpen = forceState !== undefined ? forceState : navToggle.getAttribute('aria-expanded') !== 'true';

    navToggle.setAttribute('aria-expanded', isOpen);
    navMenu.classList.toggle('is-open', isOpen);
    backdrop.classList.toggle('is-visible', isOpen);
    document.body.classList.toggle('nav-open', isOpen);

    // Focus management
    if (isOpen) {
      // Focus first link when menu opens
      const firstLink = navMenu.querySelector('a');
      if (firstLink) {
        setTimeout(() => firstLink.focus(), 100);
      }
    } else {
      // Return focus to toggle button when menu closes
      navToggle.focus();
    }
  }

  /**
   * Close menu and clean up
   */
  function closeMenu() {
    toggleMenu(false);
  }

  /**
   * Handle keyboard navigation
   * @param {KeyboardEvent} e
   */
  function handleKeyboard(e) {
    if (!navMenu.classList.contains('is-open')) return;

    switch (e.key) {
      case 'Escape':
        e.preventDefault();
        closeMenu();
        break;
      case 'Tab':
        // Trap focus within menu
        const focusableElements = navMenu.querySelectorAll('a[href]');
        const firstElement = focusableElements[0];
        const lastElement = focusableElements[focusableElements.length - 1];

        if (e.shiftKey && document.activeElement === firstElement) {
          e.preventDefault();
          lastElement.focus();
        } else if (!e.shiftKey && document.activeElement === lastElement) {
          e.preventDefault();
          firstElement.focus();
        }
        break;
    }
  }

  // Event Listeners

  // Toggle button click
  navToggle.addEventListener('click', () => toggleMenu());

  // Backdrop click to close
  backdrop.addEventListener('click', closeMenu);

  // Close menu when clicking a nav link
  navMenu.querySelectorAll('a').forEach(link => {
    link.addEventListener('click', closeMenu);
  });

  // Keyboard navigation
  document.addEventListener('keydown', handleKeyboard);

  // Handle window resize - close menu if switching to desktop
  let resizeTimer;
  window.addEventListener('resize', () => {
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(() => {
      if (window.innerWidth > 768 && navMenu.classList.contains('is-open')) {
        closeMenu();
      }
    }, 250);
  });

  // Ensure menu is closed on page load (in case of back button navigation)
  if (navMenu.classList.contains('is-open')) {
    closeMenu();
  }
})();
