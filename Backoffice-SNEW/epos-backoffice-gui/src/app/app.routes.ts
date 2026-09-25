import { Routes } from '@angular/router';

import { MainContainerComponent } from '../components/main-container/main-container.component';
import { LoginPageComponent } from '../pages/login/login-page.component';
import { authGuard } from './auth.guard';
import { globalManageGuard } from './global-manage.guard';
import { groupAdminGuard } from './group-admin.guard';

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
    path: 'data/manage-group',
    component: MainContainerComponent,
    canActivate: [authGuard, groupAdminGuard],
    data: { sectionTitle: 'Data', sectionPath: 'data', manageGroup: true }
  },
  {
    path: 'data',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Data', discoveryType: ['dataset'] }
  },
  {
    path: 'data/:resourceId/compare',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Data', discoveryType: ['dataset'], comparisonPage: true }
  },
  {
    path: 'data/:resourceId/:versionId',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Data', discoveryType: ['dataset'] }
  },
  {
    path: 'software/manage-group',
    component: MainContainerComponent,
    canActivate: [authGuard, groupAdminGuard],
    data: { sectionTitle: 'Software', sectionPath: 'software', manageGroup: true }
  },
  {
    path: 'software',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Software', discoveryType: ['software-application'] }
  },
  {
    path: 'software/:resourceId/compare',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Software', discoveryType: ['software-application'], comparisonPage: true }
  },
  {
    path: 'software/:resourceId/:versionId',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Software', discoveryType: ['software-application'] }
  },
  {
    path: 'facilities/manage-group',
    component: MainContainerComponent,
    canActivate: [authGuard, groupAdminGuard],
    data: { sectionTitle: 'Facilities', sectionPath: 'facilities', manageGroup: true }
  },
  {
    path: 'facilities',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Facilities', discoveryType: ['software-source-code'] }
  },
  {
    path: 'facilities/:resourceId/compare',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Facilities', discoveryType: ['software-source-code'], comparisonPage: true }
  },
  {
    path: 'facilities/:resourceId/:versionId',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'Facilities', discoveryType: ['software-source-code'] }
  },
  {
    path: 'memberships',
    component: MainContainerComponent,
    canActivate: [authGuard],
    data: { sectionTitle: 'My Account', accountPage: 'memberships' }
  },
  {
    path: 'global-manage',
    component: MainContainerComponent,
    canActivate: [authGuard, globalManageGuard],
    data: { sectionTitle: 'Global Administration', globalManage: true }
  }
];
