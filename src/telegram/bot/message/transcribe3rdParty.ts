/* eslint-disable no-mixed-spaces-and-tabs */
import TelegramSyncPlugin from "../../../main";
import TelegramBot from "node-telegram-bot-api";
import { getFileObject } from "./getters";
import { spawn } from 'child_process';
import { promisify } from 'util';
import * as fs from 'fs';
import { join, dirname } from 'path';
import { sendLongMessage } from "../../../utils/messageUtils";
import { DEFAULT_SETTINGS } from "../../../settings/Settings";

const readFileAsync = promisify(fs.readFile);
const unlinkAsync = promisify(fs.unlink);

function getVaultBasePath(plugin: TelegramSyncPlugin): string {
    // @ts-expect-error: basePath exists for FileSystemAdapter
    return plugin.app.vault.adapter.basePath;
}

function getTranscriptionEnv(plugin: TelegramSyncPlugin): NodeJS.ProcessEnv {
    return {
        ...process.env,
        PATH: `${process.env.PATH}:${plugin.settings.transcription.path}`
    };
}

async function getAudioFormat(inputPath: string, env: NodeJS.ProcessEnv): Promise<string> {
    const fallbackExt = 'mp3';
    return new Promise((resolve) => {
        const process = spawn('ffmpeg', [
            '-i', `"${inputPath}"`,
            '-f', 'null',
            '-'
        ], {
            shell: true,
            env: env
        });

        let stderrData = '';

        process.stderr.on('data', (data) => {
            stderrData += data.toString();
        });

        process.on('close', () => {
            const formatMatch = stderrData.match(/Input #0, ([^,]+),/);
            if (formatMatch) {
                const format = formatMatch[1].split(',')[0].toLowerCase();
                resolve(format);
            } else {
                resolve(inputPath.split('.').pop() || fallbackExt);
            }
        });

        process.on('error', () => {
            resolve(inputPath.split('.').pop() || fallbackExt);
        });
    });
}

async function checkFfmpegPresence(env: NodeJS.ProcessEnv): Promise<void> {
    return new Promise((resolve, reject) => {
        const checkFfmpeg = spawn('which', ['ffmpeg'], {
            shell: true,
            env: env
        });

        checkFfmpeg.on('close', (ffmpegCode) => {
            if (ffmpegCode !== 0) {
                reject(new Error(
                    'FFmpeg is not installed or not found in PATH. ' +
                    'Please install FFmpeg to use audio chunking for transcription. ' +
                    'You can install it via: brew install ffmpeg (macOS) or apt-get install ffmpeg (Ubuntu/Debian)'
                ));
                return;
            }
            resolve();
        });

        checkFfmpeg.on('error', (error) => {
            reject(new Error(`Failed to check for FFmpeg: ${error.message}`));
        });
    });
}

async function splitAudioFile(inputPath: string, outputDir: string, chunkDuration: number, env: NodeJS.ProcessEnv): Promise<string[]> {
    return new Promise(async (resolve, reject) => {
        try {
            await checkFfmpegPresence(env);

            const inputExt = await getAudioFormat(inputPath, env);

            const process = spawn('ffmpeg', [
                '-i', `"${inputPath}"`,
                '-f', 'segment',
                '-segment_time', chunkDuration.toString(),
                '-c', 'copy',
                `"${outputDir}/chunk_%03d.${inputExt}"`,
            ], {
                cwd: outputDir,
                shell: true,
                env: env
            });

            let stderrData = '';

            process.stderr.on('data', (data) => {
                stderrData += data.toString();
            });

            process.on('close', async (code) => {
                if (code !== 0) {
                    reject(new Error(`FFmpeg failed with code ${code}: ${stderrData}`));
                    return;
                }

                try {
                    const files = await fs.promises.readdir(outputDir);
                    const chunkFiles = files
                        .filter(file => file.startsWith('chunk_') && file.endsWith(`.${inputExt}`))
                        .sort()
                        .map(file => join(outputDir, file));

                    resolve(chunkFiles);
                } catch (error) {
                    reject(error);
                }
            });

            process.on('error', (error) => {
                reject(new Error(`Failed to execute FFmpeg: ${error.message}`));
            });
        } catch (error) {
            reject(error);
        }
    });
}

async function cleanupTempFiles(files: string[]): Promise<void> {
    for (const file of files) {
        try {
            await unlinkAsync(file);
        } catch (error) {
            console.error(`Failed to delete ${file}:`, error);
        }
    }
}

async function transcribeSingleFile(inputPath: string, outputPath: string, cwd: string, plugin: TelegramSyncPlugin): Promise<string> {
    const cmd = plugin.settings.transcription.command
        .replace('{input}', inputPath)
        .replace('{output}', outputPath);

    return new Promise((resolve, reject) => {
        const [command, ...args] = cmd.split(' ');

        const process = spawn(command, args, {
            cwd,
            env: getTranscriptionEnv(plugin),
            shell: true
        });

        let stdoutData = '';
        let stderrData = '';

        process.stdout.on('data', (data) => {
            stdoutData += data.toString();
        });

        process.stderr.on('data', (data) => {
            stderrData += data.toString();
        });

        process.on('close', async (code) => {
            if (code !== 0) {
                reject(new Error(`Transcription failed with code ${code}: ${stderrData}`));
                return;
            }

            try {
                const content = await readFileAsync(outputPath, 'utf8');
                await unlinkAsync(outputPath);
                resolve(content);
            } catch (error) {
                reject(error);
            }
        });

        process.on('error', reject);
    });
}

async function processTranscriptionCommand(filepath: string, vaultRoot: string, plugin: TelegramSyncPlugin): Promise<string> {
    const fullPath = join(vaultRoot, filepath);
    const cwd = dirname(fullPath);
    const txtPath = fullPath.replace(/\.[^/.]+$/, ".txt");

    const chunkDuration = plugin.settings.transcription.chunkDuration || DEFAULT_SETTINGS.transcription.chunkDuration;

    if (chunkDuration <= 0) {
        return await transcribeSingleFile(fullPath, txtPath, cwd, plugin);
    }

    const tempDir = join(cwd, 'temp_transcription');
    await fs.promises.mkdir(tempDir, { recursive: true });

    try {
        const chunkFiles = await splitAudioFile(fullPath, tempDir, chunkDuration, getTranscriptionEnv(plugin));

        if (chunkFiles.length === 0) {
            throw new Error('No audio chunks were created');
        }

        const transcriptions: string[] = [];
        for (const chunkFile of chunkFiles) {
            try {
                const chunkTxtPath = chunkFile.replace(/\.[^/.]+$/, ".txt");
                const transcription = await transcribeSingleFile(chunkFile, chunkTxtPath, tempDir, plugin);

                if (transcription.trim()) {
                    transcriptions.push(transcription);
                }
            } catch (error) {
                console.error('Failed to transcribe chunk:', error);
            }
        }

        return transcriptions.join('\n');

    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        if (errorMessage.includes('FFmpeg is not installed')) {
            console.warn('FFmpeg not available for chunking, attempting to transcribe the entire file instead.');
            try {
                return await transcribeSingleFile(fullPath, txtPath, cwd, plugin);
            } catch (fallbackError) {
                console.error('Fallback transcription also failed:', fallbackError);
                throw error;
            }
        }
        throw error;
    } finally {
        try {
            const files = await fs.promises.readdir(tempDir);
            const allFiles = files.map(file => join(tempDir, file));
            await cleanupTempFiles(allFiles);
            await fs.promises.rmdir(tempDir);
        } catch (error) {
            console.error('Failed to cleanup temp directory:', error);
        }
    }
}

async function sendTranscriptionReply(
    plugin: TelegramSyncPlugin,
    msg: TelegramBot.Message,
    transcription: string
): Promise<void> {
    if (!plugin.settings.transcription.replyWithTranscription || !plugin.bot) {
        return;
    }

    try {
        await sendLongMessage(plugin.bot, msg.chat.id, transcription, {
            reply_to_message_id: msg.message_id
        });
    } catch (error) {
        console.error('Failed to reply with transcription:', error);
    }
}

export async function transcribeFile(
    filePath: string,
    plugin: TelegramSyncPlugin,
    msg: TelegramBot.Message
): Promise<string> {
    if (!plugin.settings.transcription.enabled) {
        return '';
    }

    const { fileType } = getFileObject(msg);
    const supportedFileTypes = ['audio', 'voice', 'video', 'video_note'];
    if (!supportedFileTypes.includes(fileType)) {
        return '';
    }

    try {
        const transcription = await processTranscriptionCommand(
            filePath,
            getVaultBasePath(plugin),
            plugin
        );

        if (!transcription) {
            return '';
        }

        await sendTranscriptionReply(plugin, msg, transcription);
        return plugin.settings.transcription.template.replace('{text}', transcription);
    } catch (error) {
        console.error('Failed to transcribe file:', error);

        const errorMessage = error instanceof Error ? error.message : String(error);
        if (errorMessage.includes('FFmpeg is not installed')) {
            console.error('Transcription failed: FFmpeg is required for audio chunking. Please install FFmpeg or set chunk duration to 0 to disable chunking.');
        } else if (errorMessage.includes('command not found')) {
            console.error('Transcription failed: The transcription command is not found. Please check your transcription settings and ensure the command is properly configured.');
        } else {
            console.error('Transcription failed with error:', errorMessage);
        }

        return '';
    }
}
