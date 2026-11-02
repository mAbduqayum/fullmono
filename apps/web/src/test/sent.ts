/** What a request asked the api: its route ("POST /api/links"), query and JSON body. */
export async function sent(input: RequestInfo | URL | undefined) {
    // openapi-fetch hands fetch one Request, with no init.
    const request = input as Request;
    const url = new URL(request.url);
    // A clone, so that the body stays there to read again.
    const text = await request.clone().text();
    return {
        route: `${request.method} ${url.pathname}`,
        query: url.search.slice(1),
        body: text ? (JSON.parse(text) as unknown) : undefined,
    };
}
