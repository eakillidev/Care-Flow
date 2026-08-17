import { Component, Input } from '@angular/core';
import { AuthService } from '../auth/auth.service';

@Component({ selector: 'app-header', templateUrl: './app-header.html', styleUrl: './app-header.css' })
export class AppHeader {
  @Input({ required: true }) section = '';
  constructor(private readonly auth: AuthService) {}
  logout(): void { this.auth.logout(); }
}
