export interface ShareApi {

  saveConfigurables(value: string, title?: string, description?: string): Promise<string>;

  retrieveConfigurables(key: string): Promise<string | null>;
}
