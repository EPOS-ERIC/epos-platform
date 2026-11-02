/*
         Copyright 2021 EPOS ERIC

 Licensed under the Apache License, Version 2.0 (the License); you may not
 use this file except in compliance with the License.  You may obtain a copy
 of the License at

   http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an AS IS BASIS, WITHOUT
 WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
 License for the specific language governing permissions and limitations under
 the License.
 */
 import { BaseUrl } from 'api/webApi/classes/baseurl.interface';
 import { Rest } from 'api/webApi/classes/rest.interface';
 import { BackOfficeUserApi } from 'api/webApi/classes/backOfficeUser.interface';
 import { SimpleBackOfficeUser } from 'api/webApi/data/impl/simpleBackofficeUser';

 export class DevBackofficeUserApi implements BackOfficeUserApi {

   constructor(private readonly baseUrl: BaseUrl, private readonly rest: Rest) { }

    async getBackOfficeUser(_self: string = 'self'): Promise<SimpleBackOfficeUser>{
        const builder = this.baseUrl.urlBuilder();
        builder.addPathElements('users', 'self');
        const url = builder.build();

        const userInfo = await this.rest.get(url) as {
          id: string;
          email: string;
          given_name: string;
          memberships: unknown[];
          global_roles: string[];
        };

        if (!userInfo) {
            throw new Error('No backOffice User details found');
        }
        if (!userInfo.id || !userInfo.email) {
            throw new Error('BackOffice User response is missing id or email');
        }

        const groups = [...(userInfo.memberships ?? []), ...(userInfo.global_roles ?? [])];
        return SimpleBackOfficeUser.make(userInfo.id, userInfo.email, userInfo.given_name || userInfo.email, groups);

    }
 }
