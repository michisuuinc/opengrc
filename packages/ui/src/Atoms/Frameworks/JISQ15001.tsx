// Copyright (c) 2026 MICHISUU Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// A plain text badge for JIS Q 15001 (the standard behind the Japanese
// PrivacyMark). It deliberately avoids the PrivacyMark logo, which is a
// registered trademark of JIPDEC.
export function JISQ15001(props: { className?: string }) {
  const { className } = props;

  return (
    <svg
      className={className}
      viewBox="0 0 60 60"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="30" cy="30" r="27.5" className="stroke-txt-primary" />
      <text
        x="30"
        y="28"
        textAnchor="middle"
        fontSize="13"
        fontWeight="700"
        className="fill-txt-primary"
      >
        JIS Q
      </text>
      <text
        x="30"
        y="42"
        textAnchor="middle"
        fontSize="11"
        className="fill-txt-primary"
      >
        15001
      </text>
    </svg>
  );
}
