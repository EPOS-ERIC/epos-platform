/* eslint-disable @typescript-eslint/no-unsafe-call */
/* eslint-disable @typescript-eslint/no-unsafe-member-access */
import moment from 'moment-es6';
import { InformationsService } from 'services/informationsService.service';
import { PoliciesService } from 'services/policiesService.service';
import { GNSS_STATIONS_WITH_PRODUCTS, Service } from './constants';

// ***********************************************
// This example namespace declaration will help
// with Intellisense and code completion in your
// IDE or Text Editor.
// ***********************************************
// declare namespace Cypress {
//   interface Chainable<Subject = any> {
//     customCommand(param: any): typeof customCommand;
//   }
// }
//
// function customCommand(param: any): void {
//   console.warn(param);
// }
//
// NOTE: You can use it like so:
// Cypress.Commands.add('customCommand', customCommand);
//
// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add("login", (email, password) => { ... })
//
//
// -- This is a child command --
// Cypress.Commands.add("drag", { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add("dismiss", { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite("visit", (originalFn, url, options) => { ... })
//


// *********** TYPE DECLARATIONS IN commands.d.ts file ***************/

// // -- Accept policies
Cypress.Commands.add('policyAccept', () => {
  // PoliciesService.storeConsentsTimestamp(moment());
  // PoliciesService.storeCookieConsent(true);

  cy.getByDataCy('toggle-terms-checkbox').find('.mat-checkbox-inner-container').click();
  cy.getByDataCy('toggle-privacy-checkbox').find('.mat-checkbox-inner-container').click();
  cy.getByDataCy('accept-terms-button').click();

  cy.contains('EPOS POLICIES').should('not.exist');
});

Cypress.Commands.add('newFeatures', () => {
  cy.get('.new-feature-popup .epos-close-btn').click();
  cy.contains('NEW FEATURES').should('not.exist');
});

Cypress.Commands.add('policyAcceptAndWelcomePopup', () => {
  cy.visit('/');
  cy.policyAccept();
  cy.newFeatures();
});

// Command to do get('[data-cy="..."') in a more readable way
// Example: cy.getByDataCy('domain-list > #list').first().click();
Cypress.Commands.add('getByDataCy', (args) => {
  // Get the first element of the string
  const dataCy = args.split(' > ')[0];
  const string = `[data-cy="${dataCy}"] ${args.replace(dataCy, '')}`;
  return cy.get(string);
});

Cypress.Commands.add('init', () => {
  const concepts = [
    { id: '00000000-0000-4000-8000-000000000001', name: 'DATA' },
    { id: '00000000-0000-4000-8000-000000000002', name: 'FACILITIES' },
    { id: '00000000-0000-4000-8000-000000000003', name: 'SOFTWARE' },
  ];
  cy.intercept('GET', '**/api/v1/concepts', concepts);
  cy.intercept('GET', '**/api/v1/search*', (request) => {
    const concept = concepts.find(item => item.id === request.query.conceptId);
    if (concept === undefined) {
      throw new Error(`Unexpected search conceptId: ${String(request.query.conceptId)}`);
    }
    request.alias = `initial${concept.name}Search`;
    request.reply(concept.name === 'DATA' ? { fixture: 'distributions.json' } : []);
  });
  cy.intercept('GET', '**/api/v1/groups', { fixture: 'groups.json' });
  cy.fixture('categories.json').then((categories: Record<string, unknown[]>) => {
    cy.intercept('GET', /\/api\/v1\/groups\/[^/]+\/categories$/, (request) => {
      request.reply(categories[request.url.split('/').slice(-2)[0]]);
    });
  });
  cy.intercept('GET', /testpath\/api\/v1\/organizations(?:\?.*)?$/, { fixture: 'organizations.json' }).as('organizations');
  cy.policyAcceptAndWelcomePopup();
  cy.wait(['@organizations']);
  cy.wait(['@initialDATASearch', '@initialFACILITIESSearch', '@initialSOFTWARESearch']);
  cy.getByDataCy('domain-list').should('be.visible');
});

Cypress.Commands.add('freeTextSearch', (text: string) => {
  // Intercept the request to filter the results
  cy.intercept({ method: 'GET', pathname: '**/api/v1/search', query: { q: text } }).as('search');

  // Search for the test service
  cy.getByDataCy('search-panel-free-text-input').type(text + '{enter}');

  // Wait for the list to be filtered
  cy.wait('@search');
});

Cypress.Commands.add('getLeafletPane', (id: string) => {
  return cy.get('.leaflet-' + id + '-pane');
});

Cypress.Commands.add('getServiceMapFeatures', (service: Service) => {
  return service.getMarkerFunction(service);
});

// Search for a specific service and show only it
Cypress.Commands.add('searchForService', (service: Service) => {
  // Intercept the search request
  cy.intercept(
    'GET', /\/api\/v1\/search\?/,
    { fixture: service.searchJson() });

  // Search for the fake distribution
  cy.freeTextSearch(service.name);
});

Cypress.Commands.add('interceptService', (service: Service) => {
  // Intercept all the possible request for this service, stubbing them with the correct fixture and save a reference to the request to be able to wait on it
  // Details belong to the dataset version returned by discovery.
  cy.fixture(service.searchJson()).then((datasets: Array<{ resource_id: string; version_id: string }>) => {
    const dataset = datasets[0];
    cy.intercept('GET', `**/resources/dataset/${dataset.resource_id}/versions/${dataset.version_id}`,
      { fixture: service.detailsJson() }).as(service.detailsRequest.substring(1));
  });
  cy.intercept('POST', `**/distributions/${service.id}/access*`, (request) => {
    const parameters = request.body.parameters as Array<{ value: unknown }>;
    const hasBoundingBox = [-5.00977, 54.00777, 9.93164, 39.02772].every(value =>
      parameters.some(parameter => Number(parameter.value).toFixed(5) === value.toFixed(5)));
    request.alias = hasBoundingBox ? service.bboxFilteredRequest.substring(1) : service.dataRequest.substring(1);
    request.reply({ fixture: hasBoundingBox ? service.bboxFilteredJson() : service.dataJson() });
  });
  cy.intercept({ method: 'GET', pathname: '**/api/v1/search', query: { q: service.name } },
    { fixture: service.searchJson() }).as(service.searchRequest.substring(1));
  // Prepare the provider URL only when a scenario requests it.
  cy.intercept(
    'POST',
    `**/distributions/${service.id}/prepare*`,
    request => request.reply({ fixture: service.getOriginalURL() })
  ).as(service.getOriginalUrlRequest.substring(1));
});
