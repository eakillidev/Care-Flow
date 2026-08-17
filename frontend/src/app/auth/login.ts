import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { finalize } from 'rxjs';
import { AuthService } from './auth.service';

@Component({ selector: 'app-login', imports: [FormsModule], templateUrl: './login.html', styleUrl: './login.css' })
export class Login {
  email = '';
  password = '';
  loading = false;
  error = '';

  constructor(private readonly auth: AuthService, private readonly router: Router) {}

  submit(): void {
    if (this.loading) return;
    if (!this.email.trim() || !this.password) {
      this.error = 'Email and password are required.';
      return;
    }
    this.loading = true;
    this.error = '';
    this.auth.login(this.email, this.password).pipe(finalize(() => this.loading = false)).subscribe({
      next: ({ user }) => void this.router.navigateByUrl(user.role === 'coordinator' ? '/coordinator' : '/caregiver'),
      error: () => this.error = 'Unable to sign in with those credentials.'
    });
  }
}
