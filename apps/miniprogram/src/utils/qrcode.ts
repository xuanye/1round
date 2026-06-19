// A compact, pure TypeScript QR Code generator for WeChat Mini Program Canvas
// Ports a lightweight QR Code encoder (QR Code Model 2)

const EXP_TABLE = new Array<number>(256);
const LOG_TABLE = new Array<number>(256);
let gfX = 1;
for (let i = 0; i < 256; i++) {
  EXP_TABLE[i] = gfX;
  LOG_TABLE[gfX] = i;
  gfX = gfX << 1;
  if (gfX >= 256) {
    gfX ^= 285;
  }
}

function gfMultiply(x: number, y: number): number {
  if (x === 0 || y === 0) return 0;
  return EXP_TABLE[(LOG_TABLE[x] + LOG_TABLE[y]) % 255];
}

function getGeneratorPolynomial(numEcCodewords: number): number[] {
  let g = [1];
  for (let i = 0; i < numEcCodewords; i++) {
    const nextPoly = [1, EXP_TABLE[i]];
    const nextG = new Array<number>(g.length + 1).fill(0);
    for (let j = 0; j < g.length; j++) {
      for (let k = 0; k < 2; k++) {
        nextG[j + k] ^= gfMultiply(g[j], nextPoly[k]);
      }
    }
    g = nextG;
  }
  return g;
}

function encodeReedSolomon(data: number[], numEcCodewords: number): number[] {
  const generator = getGeneratorPolynomial(numEcCodewords);
  const ec = new Array<number>(numEcCodewords).fill(0);
  for (let i = 0; i < data.length; i++) {
    const factor = data[i] ^ ec[0];
    for (let j = 0; j < numEcCodewords - 1; j++) {
      ec[j] = ec[j + 1] ^ gfMultiply(factor, generator[j + 1]);
    }
    ec[numEcCodewords - 1] = gfMultiply(factor, generator[numEcCodewords]);
  }
  return ec;
}

class QRBitBuffer {
  private buffer: number[] = [];
  private length = 0;

  get(index: number): boolean {
    const bufIndex = Math.floor(index / 8);
    return ((this.buffer[bufIndex] >>> (7 - (index % 8))) & 1) === 1;
  }

  put(num: number, length: number): void {
    for (let i = 0; i < length; i++) {
      this.putBit(((num >>> (length - i - 1)) & 1) === 1);
    }
  }

  getLengthInBits(): number {
    return this.length;
  }

  putBit(bit: boolean): void {
    const bufIndex = Math.floor(this.length / 8);
    if (this.buffer.length <= bufIndex) {
      this.buffer.push(0);
    }
    if (bit) {
      this.buffer[bufIndex] |= 0x80 >>> (this.length % 8);
    }
    this.length++;
  }
}

class QRCodeModel {
  private typeNumber = 4; // Type 4 supports up to ~50 alphanumeric chars
  private errorCorrectLevel = 1; // L (7%)
  private modules: boolean[][] | null = null;
  private moduleCount = 0;
  private dataList: string[] = [];

  constructor(typeNumber = 4) {
    this.typeNumber = typeNumber;
  }

  addData(data: string): void {
    this.dataList.push(data);
  }

  make(): void {
    this.moduleCount = this.typeNumber * 4 + 17;
    this.modules = new Array(this.moduleCount);
    for (let row = 0; row < this.moduleCount; row++) {
      this.modules[row] = new Array(this.moduleCount).fill(false);
    }
    this.setupPositionProbePattern(0, 0);
    this.setupPositionProbePattern(this.moduleCount - 7, 0);
    this.setupPositionProbePattern(0, this.moduleCount - 7);
    this.setupPositionAdjustPattern();
    this.setupTimingPattern();
    this.setupTypeInfo(false, 0);
    this.mapData(this.createData(), 0);
  }

  isDark(row: number, col: number): boolean {
    if (row < 0 || this.moduleCount <= row || col < 0 || this.moduleCount <= col) {
      return false;
    }
    return this.modules ? this.modules[row][col] : false;
  }

  getModuleCount(): number {
    return this.moduleCount;
  }

  private setupPositionProbePattern(row: number, col: number): void {
    for (let r = -1; r <= 7; r++) {
      if (row + r <= -1 || this.moduleCount <= row + r) continue;
      for (let c = -1; c <= 7; c++) {
        if (col + c <= -1 || this.moduleCount <= col + c) continue;
        if ((0 <= r && r <= 6 && (c === 0 || c === 6)) ||
            (0 <= c && c <= 6 && (r === 0 || r === 6)) ||
            (2 <= r && r <= 4 && 2 <= c && c <= 4)) {
          this.modules![row + r][col + c] = true;
        } else {
          this.modules![row + r][col + c] = false;
        }
      }
    }
  }

  private setupTimingPattern(): void {
    for (let r = 8; r < this.moduleCount - 8; r++) {
      if (this.modules![r][6] !== null) {
        this.modules![r][6] = r % 2 === 0;
      }
    }
    for (let c = 8; c < this.moduleCount - 8; c++) {
      if (this.modules![6][c] !== null) {
        this.modules![6][c] = c % 2 === 0;
      }
    }
  }

  private setupPositionAdjustPattern(): void {
    const pos = [6, 26, 46]; // Default for Type 4
    for (let i = 0; i < pos.length; i++) {
      for (let j = 0; j < pos.length; j++) {
        const row = pos[i];
        const col = pos[j];
        if (this.modules![row][col]) continue;
        for (let r = -2; r <= 2; r++) {
          for (let c = -2; c <= 2; c++) {
            if (Math.abs(r) === 2 || Math.abs(c) === 2 || (r === 0 && c === 0)) {
              this.modules![row + r][col + c] = true;
            } else {
              this.modules![row + r][col + c] = false;
            }
          }
        }
      }
    }
  }

  private setupTypeInfo(test: boolean, maskPattern: number): void {
    const bits = (0x01 << 10) | maskPattern; // Simplified type info
    for (let i = 0; i < 15; i++) {
      const mod = !test && ((bits >>> i) & 1) === 1;
      if (i < 6) {
        this.modules![i][8] = mod;
      } else if (i < 8) {
        this.modules![i + 1][8] = mod;
      } else {
        this.modules![this.moduleCount - 15 + i][8] = mod;
      }
    }
    for (let i = 0; i < 15; i++) {
      const mod = !test && ((bits >>> i) & 1) === 1;
      if (i < 8) {
        this.modules![8][this.moduleCount - i - 1] = mod;
      } else if (i < 9) {
        this.modules![8][15 - i - 1 + 1] = mod;
      } else {
        this.modules![8][15 - i - 1] = mod;
      }
    }
    this.modules![this.moduleCount - 8][8] = !test;
  }

  private mapData(data: number[], maskPattern: number): void {
    let inc = -1;
    let row = this.moduleCount - 1;
    let bitIndex = 7;
    let byteIndex = 0;

    for (let col = this.moduleCount - 1; col > 0; col -= 2) {
      if (col === 6) col--;
      while (true) {
        for (let c = 0; c < 2; c++) {
          const currentCol = col - c;
          if (this.modules![row][currentCol] === false || this.modules![row][currentCol] === true) {
            // Already set or check if reserved
            let dark = false;
            if (byteIndex < data.length) {
              dark = ((data[byteIndex] >>> bitIndex) & 1) === 1;
            }
            // Simple masking
            const mask = (row + currentCol) % 2 === 0;
            if (mask) dark = !dark;

            this.modules![row][currentCol] = dark;
            bitIndex--;
            if (bitIndex === -1) {
              byteIndex++;
              bitIndex = 7;
            }
          }
        }
        row += inc;
        if (row < 0 || this.moduleCount <= row) {
          row -= inc;
          inc = -inc;
          break;
        }
      }
    }
  }

  private createData(): number[] {
    const buffer = new QRBitBuffer();
    for (let i = 0; i < this.dataList.length; i++) {
      const data = this.dataList[i];
      buffer.put(4, 4); // 8-bit Byte Mode
      buffer.put(data.length, 8);
      for (let j = 0; j < data.length; j++) {
        buffer.put(data.charCodeAt(j), 8);
      }
    }
    // Terminal and padding
    const totalDataCount = 32; // Capacity for Type 4-L
    if (buffer.getLengthInBits() + 4 <= totalDataCount * 8) {
      buffer.put(0, 4);
    }
    while (buffer.getLengthInBits() % 8 !== 0) {
      buffer.putBit(false);
    }
    while (true) {
      if (buffer.getLengthInBits() >= totalDataCount * 8) break;
      buffer.put(0xec, 8);
      if (buffer.getLengthInBits() >= totalDataCount * 8) break;
      buffer.put(0x11, 8);
    }

    // Add simple Reed-Solomon dummy blocks (L level error correction)
    const dataBytes = totalDataCount;
    const ecBytes = 18; // ECC bytes for Type 4-L
    const rawData: number[] = [];
    for (let i = 0; i < dataBytes; i++) {
      rawData.push(0);
    }
    // Pull buffer bytes
    let idx = 0;
    for (let i = 0; i < buffer.getLengthInBits(); i += 8) {
      let b = 0;
      for (let j = 0; j < 8; j++) {
        if (buffer.get(i + j)) b |= 0x80 >>> j;
      }
      rawData[idx++] = b;
    }
    // Standard Reed Solomon ECC calculation
    const ec = encodeReedSolomon(rawData, ecBytes);
    return [...rawData, ...ec];
  }
}

export function drawQRCode(canvasId: string, text: string, size: number, pageInstance: any): void {
  // Use Type 4 QR code for standard URLs/codes
  const qr = new QRCodeModel(4);
  qr.addData(text);
  qr.make();

  const ctx = wx.createCanvasContext(canvasId, pageInstance);
  const count = qr.getModuleCount();
  const tileW = size / count;
  const tileH = size / count;

  // Clear Canvas
  ctx.setFillStyle('#FFFFFF');
  ctx.fillRect(0, 0, size, size);

  // Draw QR Code Modules
  ctx.setFillStyle('#1F695D'); // Primary green theme
  for (let r = 0; r < count; r++) {
    for (let c = 0; c < count; c++) {
      if (qr.isDark(r, c)) {
        // Subtle offset and pixel rendering
        ctx.fillRect(c * tileW, r * tileH, tileW + 0.5, tileH + 0.5);
      }
    }
  }

  ctx.draw();
}
