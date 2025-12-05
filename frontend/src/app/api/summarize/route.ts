import { NextRequest, NextResponse } from 'next/server';
import https from 'https';
import pdfParse from 'pdf-parse';
import mammoth from 'mammoth';

const GEMINI_API_KEY = 'AIzaSyAqL_bB-W5koR2EZSkQhXfqiaNjtyuaXQ8';
const GEMINI_API_URL = 'https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent';

// Helper function to make HTTPS request (handles SSL issues in Docker)
async function makeHttpsRequest(url: string, options: { method: string; headers: Record<string, string>; body: string }): Promise<{ ok: boolean; status: number; json: () => Promise<any> }> {
  return new Promise((resolve, reject) => {
    const urlObj = new URL(url);
    const reqOptions = {
      hostname: urlObj.hostname,
      port: 443,
      path: urlObj.pathname + urlObj.search,
      method: options.method,
      headers: options.headers,
      rejectUnauthorized: false, // Skip SSL verification for Docker environment
    };

    const req = https.request(reqOptions, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        resolve({
          ok: res.statusCode !== undefined && res.statusCode >= 200 && res.statusCode < 300,
          status: res.statusCode || 500,
          json: async () => JSON.parse(data),
        });
      });
    });

    req.on('error', reject);
    req.write(options.body);
    req.end();
  });
}

interface GeminiResponse {
  candidates?: {
    content?: {
      parts?: {
        text?: string;
      }[];
    };
  }[];
  error?: {
    message: string;
  };
}

async function extractTextFromBinary(mimeType: string, base64Data: string): Promise<string> {
  const buffer = Buffer.from(base64Data, 'base64');

  if (mimeType === 'application/pdf') {
    try {
      const pdfData = await pdfParse(buffer);
      return pdfData.text;
    } catch (error) {
      console.error('PDF parsing error:', error);
      throw new Error('Failed to extract text from PDF');
    }
  }

  if (mimeType === 'application/vnd.openxmlformats-officedocument.wordprocessingml.document') {
    try {
      const result = await mammoth.extractRawText({ buffer });
      return result.value;
    } catch (error) {
      console.error('DOCX parsing error:', error);
      throw new Error('Failed to extract text from DOCX');
    }
  }

  throw new Error(`Unsupported binary format: ${mimeType}`);
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    let { text } = body;

    if (!text || typeof text !== 'string') {
      return NextResponse.json(
        { error: 'Document text is required' },
        { status: 400 }
      );
    }

    // Check if this is a binary document that needs extraction
    if (text.startsWith('__BINARY_DOCUMENT__:')) {
      const parts = text.split(':');
      if (parts.length >= 3) {
        const mimeType = parts[1];
        const base64Data = parts.slice(2).join(':'); // Handle base64 that might contain colons
        try {
          text = await extractTextFromBinary(mimeType, base64Data);
        } catch (error: any) {
          return NextResponse.json(
            { error: error.message || 'Failed to extract document text' },
            { status: 400 }
          );
        }
      }
    }

    if (!text.trim()) {
      return NextResponse.json(
        { error: 'Document appears to be empty or could not be read' },
        { status: 400 }
      );
    }

    // Truncate text if too long (Gemini has context limits)
    const maxLength = 30000;
    const truncatedText = text.length > maxLength
      ? text.substring(0, maxLength) + '\n\n[Document truncated due to length...]'
      : text;

    const prompt = `Summarize the following document in 3-5 concise bullet points:\n\n${truncatedText}`;

    const response = await makeHttpsRequest(`${GEMINI_API_URL}?key=${GEMINI_API_KEY}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        contents: [
          {
            parts: [
              {
                text: prompt,
              },
            ],
          },
        ],
        generationConfig: {
          temperature: 0.3,
          maxOutputTokens: 1024,
        },
      }),
    });

    if (!response.ok) {
      const errorData = await response.json();
      console.error('Gemini API error:', errorData);
      return NextResponse.json(
        { error: errorData.error?.message || 'Failed to generate summary' },
        { status: response.status }
      );
    }

    const data: GeminiResponse = await response.json();

    const summary = data.candidates?.[0]?.content?.parts?.[0]?.text;

    if (!summary) {
      return NextResponse.json(
        { error: 'No summary generated' },
        { status: 500 }
      );
    }

    return NextResponse.json({ summary });
  } catch (error) {
    console.error('Summarize API error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
