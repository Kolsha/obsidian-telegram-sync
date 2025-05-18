/* eslint-disable no-mixed-spaces-and-tabs */
import TelegramSyncPlugin from "../../../main";
import TelegramBot from "node-telegram-bot-api";
import { getFileObject } from "./getters";
import { spawn } from 'child_process';
import { promisify } from 'util';
import * as fs from 'fs';
import { join, dirname } from 'path';

const readFileAsync = promisify(fs.readFile);
const unlinkAsync = promisify(fs.unlink);

async function processTranscriptionCommand(filepath: string, vaultRoot: string, plugin: TelegramSyncPlugin): Promise<string> {
    const fullPath = join(vaultRoot, filepath);
    const CWD = dirname(fullPath);
    const txtPath = fullPath.replace(/\.[^/.]+$/, ".txt");
    
    // Prepare command by replacing placeholders
    const cmd = plugin.settings.transcription.command
        .replace('{input}', fullPath)
        .replace('{output}', txtPath);

    // Prepare environment with custom PATH
    const env = {
        ...process.env,
        PATH: `${process.env.PATH}:${plugin.settings.transcription.path}`
    };

    return new Promise((resolve, reject) => {
        // Split command into array for spawn
        const [command, ...args] = cmd.split(' ');
        
        const process = spawn(command, args, {
            cwd: CWD,
            env,
            shell: true
        });

        let stdoutData = '';
        let stderrData = '';

        process.stdout.on('data', (data) => {
            const output = data.toString();
            stdoutData += output;
            console.log('Transcription output:', output);
        });

        process.stderr.on('data', (data) => {
            const output = data.toString();
            stderrData += output;
            console.log('Transcription error:', output);
        });

        process.on('close', async (code) => {
            if (code !== 0) {
                reject(new Error(`Transcription process exited with code ${code}\nStderr: ${stderrData}`));
                return;
            }

            try {
                const content = await readFileAsync(txtPath, 'utf8');
                await unlinkAsync(txtPath);
                resolve(content);
            } catch (error) {
                reject(error);
            }
        });

        process.on('error', (error) => {
            reject(error);
        });
    });
}

export async function transcribeFile(
    filePath: string,
    plugin: TelegramSyncPlugin,
    msg: TelegramBot.Message
): Promise<string> {
    // Check if transcription is enabled
    if (!plugin.settings.transcription.enabled) {
        return '';
    }

    // Check if it's an audio or voice message
    const { fileType } = getFileObject(msg);
    if (fileType != 'audio' && fileType != 'voice') {
        return '';
    }

    try {
        const transcription = await processTranscriptionCommand(
            filePath, 
            plugin.app.vault.adapter.getBasePath(), 
            plugin
        );
        console.log('Transcribed text:', transcription);
        if (!transcription) {
            return '';
        }
        
        // Reply with transcription if enabled
        if (plugin.settings.transcription.replyWithTranscription && plugin.bot) {
            try {
                await plugin.bot.sendMessage(msg.chat.id, transcription, {
                    reply_to_message_id: msg.message_id
                });
            } catch (error) {
                console.error('Failed to reply with transcription:', error);
            }
        }
        
        return plugin.settings.transcription.template.replace('{text}', transcription);
    } catch (error) {
        console.error('Failed to transcribe file:', error);
        return '';
    }
}
