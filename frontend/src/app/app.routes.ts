import { Routes } from '@angular/router';
import { Home } from './home/home';
import { CoordinatorDashboard } from './coordinator/coordinator-dashboard';
import { Login } from './auth/login';
import { guestGuard, roleGuard } from './auth/auth.guard';
import { CaregiverDashboard } from './caregiver/caregiver-dashboard';

export const routes: Routes = [
  {
    path: '',
    component: Home,
    title: 'CareFlow'
  },
  {
    path: 'login',
    component: Login,
    canActivate: [guestGuard],
    title: 'Sign In | CareFlow'
  },
  {
    path: 'coordinator',
    component: CoordinatorDashboard,
    canActivate: [roleGuard('coordinator')],
    title: 'Coordinator Dashboard | CareFlow'
  },
  {
    path: 'caregiver',
    component: CaregiverDashboard,
    canActivate: [roleGuard('caregiver')],
    title: 'My Visits | CareFlow'
  }
];
