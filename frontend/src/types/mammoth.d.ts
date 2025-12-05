declare module 'mammoth' {
  interface ConversionResult {
    value: string;
    messages: Message[];
  }

  interface Message {
    type: string;
    message: string;
  }

  interface Options {
    buffer?: Buffer;
    path?: string;
    arrayBuffer?: ArrayBuffer;
  }

  export function extractRawText(options: Options): Promise<ConversionResult>;
  export function convertToHtml(options: Options): Promise<ConversionResult>;
  export function convertToMarkdown(options: Options): Promise<ConversionResult>;
}
