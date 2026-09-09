import { Routes } from '@angular/router';

import { LoginPageComponent } from '../components/login-page/login-page.component';
import { MainContainerComponent } from '../components/main-container/main-container.component';
import { authGuard } from './auth.guard';

export const routes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    redirectTo: 'login'
  },
  {
    path: 'login',
    component: LoginPageComponent
  },
  {
    path: 'last-page-redirect',
    component: LoginPageComponent
  },
  {
    path: 'testpath/last-page-redirect',
    component: LoginPageComponent
  },
  {
    path: 'data',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Data', discoveryType: ['dataset'] }
  },
  {
    path: 'data/:resourceId/:versionId',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Data', discoveryType: ['dataset'] }
  },
  {
    path: 'software',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Software', discoveryType: ['software-application'] }
  },
  {
    path: 'software/:resourceId/:versionId',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Software', discoveryType: ['software-application'] }
  },
  {
    path: 'facilities',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Facilities', discoveryType: ['software-source-code'] }
  },
  {
    path: 'facilities/:resourceId/:versionId',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Facilities', discoveryType: ['software-source-code'] }
  },
  {
    path: 'admin/groups',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Groups', adminPage: 'groups' }
  },
  {
    path: 'admin/categories',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Categories', adminPage: 'categories' }
  },
  {
    path: 'admin/organizations',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Organizations', adminPage: 'organizations' }
  }
];
