import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthService } from './auth.service';
import { Login } from './login';

describe('Login', () => {
  let fixture: ComponentFixture<Login>;
  const auth = { login: vi.fn() };
  const router = { navigateByUrl: vi.fn() };
  beforeEach(async () => {
    auth.login.mockReset(); router.navigateByUrl.mockReset();
    await TestBed.configureTestingModule({ imports: [Login], providers: [{ provide: AuthService, useValue: auth }, { provide: Router, useValue: router }] }).compileComponents();
    fixture = TestBed.createComponent(Login);
  });
  it.each([['coordinator', '/coordinator'], ['caregiver', '/caregiver']] as const)('redirects a valid %s login', (role, route) => {
    auth.login.mockReturnValue(of({ token: 'jwt', user: { role } }));
    fixture.componentInstance.email = 'user@example.com'; fixture.componentInstance.password = 'password'; fixture.componentInstance.submit();
    expect(router.navigateByUrl).toHaveBeenCalledWith(route);
  });
  it('shows a safe failed-login message', () => {
    auth.login.mockReturnValue(throwError(() => new Error('database detail')));
    fixture.componentInstance.email = 'user@example.com'; fixture.componentInstance.password = 'wrong'; fixture.componentInstance.submit(); fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Unable to sign in');
    expect(fixture.nativeElement.textContent).not.toContain('database detail');
  });
});
