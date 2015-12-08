#!/usr/bin/env node
try {
	require('shelljs/global')
} catch(e) {
	console.log('Error: Unable to find module shelljs.\nRun "export NODE_PATH=\'/usr/local/lib/node_modules\'", then try again.')
	process.exit()
}


// = = = = = = = = = = = =   Go! -- Git Optimization.  Where the magic happens   = = = = = = = = = = = = //
var Go = function(args) {
	this.args = args
	this.init()
}
Go.prototype = {
	
	init: function() {
		if (!this.args.length) {
			this.options.evaluate()
			exit(0)
		}
		
		this.getOptions()
	},
	
	// Get all of the passed-in options and their params and execute them
	getOptions: function() {
		var nextArg
		while (nextArg = this.args.shift()) {
			this.options.evaluate(nextArg.slice(1), this.params)
		}
	},
	
	get params() {
		var params = []
		while (this.args.length && !(this.args[0].slice(0, 1) === '-' && this.opts.indexOf(this.args[0].slice(1)) > -1 && this.args[0].length === 2)) {
			params.push(this.args.shift())
		}
		return params
	},
	
	opts: ['d', 'D', 'h', 'l', 'm', 'n', 'r', 's', 't', 'u'],
	
	options: {
		evaluate: function(option, params) {
			switch (option) {
				case 'd':
					this.delete()
					break
				case 'D':
					this.delete(true)
					break
				case 'h':
					this.help()
					break
				case 'l':
					this.list()
					break
				case 'm':
					this.commit(params.join(' '))
					break
				case 'n':
					this.new(params.join('_'))
					break
				case 'r':
					this.request(params.join(' '))
					break
				case 's':
					this.save(params.join('_'))
					break
				case 't':
					this.test(params)
					break
				case 'u':
					this.update()
					break
				default:
					this.commit()
			}
		},
		
		commit: function(message) {
			assertOnLocal()
			assertChanges()
			message || (message = getCommitMessage())
			var branch = currentBranch()
			
			echo('Adding files to commit')
			e('git add --all')
			echo('Committing')
			e('git commit -m "' + message + '"')
			echo('Merging into dev branch')
			e('git checkout dev && git pull && git merge ' + branch + ' -Xignore-space-change -X theirs')
			echo('Pushing dev branch')
			e('git push')
			echo('Checking out "' + branch + '" again')
			e('git checkout ' + branch)
			
			e('git config --local go.' + branch + '.commitnum $((`git config --local go.' + branch + '.commitnum` + 1))')
			echo('Success!  All systems go.')
		},
		
		delete: function(hard) {
			switchToLocal()
			if (hasChanges()) {
				if (hard) {
					e('git checkout -- .')
				} else {
					echo('Couldn\'t delete local branch; you have uncommitted changes')
					echo('Run "go [-m [message]]" to commit your changes.  Or run go -D to force delete')
					return
				}
			}
			var branch = currentBranch()
			prompt('Branch to be deleted: ' + branch + '.  Proceed [y/n]?', function(input) {
				if (!/[Yy]/.test(input)) {
					echo('Exiting...')
					exit(0)
				} else {
					e('git checkout master && git pull') // Switch to master
					try {
						echo('Deleting the local branch')
						e_throw('git branch -d ' + branch) // Delete local branch
						echo('Deleting the remote branch')
						e_throw('git push origin :' + branch) // Delete upstream branch
					} catch(exception) {
						echo('Exception thrown: ', exception)
						echo('Kurt must have deleted the upstream branch... Cleaning up Kurt\'s generosity...')
						e('git fetch -p')
					}
					echo('Cleaning up')
					e('git config --local --unset go.' + branch + '.commitnum && git config --local --unset go.' + branch + '.haspullrequest')
					echo('Success!  Branch "' + branch + '" successfully deleted')
					exit(0)
				}
			})
		},
		
		help: function() {
			echo(`USAGE: go [opt]... [optarg]...
Automatically synchronize local development branch with a remote dev server.

  -d		Delete your current local branch
  -D 		Force delete your current local branch
  -h 		Display this help
  -l		List local branches
  -m 		Supply a custom message for the commit
  -n [name]	Create new branch and set its upstream target
  -r		Create a pull request in github for your current local branch
  -s [branch]	Save me!  I made changes on the wrong branch.  Switch changes to [branch]
  -u 		Update the master and dev branches of this repo.`)
		},
		
		list: function() {
			exec('git rev-parse --abbrev-ref --branches | grep -P "^((?!dev)(?!master).)"').output.trim()
		},
		
		new: function(branchName) {
			if (!branchName) {
				echo('A name is required for the new branch.  None given.  Exiting...')
				exit(1)
			}
			echo('Checking out master')
			e('git checkout master')
			echo('Updating master')
			e('git pull')
			echo('Creating new branch "' + branchName + '" and switching to it')
			e('git checkout -b ' + branchName)
			
			if (!onLocal()) {
				echo('Error: unable to create branch ' + branchName + '.  Name might be invalid.  Exiting...')
				exit(1)
			}
			
			echo('Setting upstream target to "origin/' + branchName + '"')
			e('git push origin ' + branchName + ' && git branch --set-upstream-to origin/' + branchName)
			createGitConfig()
			echo('Success!  You are safely on your new branch')
		},
		
		request: function(requestMessage) {
			requestMessage || (requestMessage = 'Ohdajoiza beinKurtz')
			switchToLocal()
			echo('Creating pull request for branch "' + currentBranch() + '"')
			try {
				e('git push')
				var repo = e("git remote show origin | grep 'Fetch URL' | cut -d'/' -f 5 | cut -d'.' -f 1"),
					branch = currentBranch(),
					data = JSON.stringify({
						title: branch,
						head: branch,
						base: 'master',
						body: requestMessage
					})
				
				var result = e('curl -X POST -d \'' + data + '\' -H "Authorization: token `git config --global user.token`" https://api.github.com/repos/ipartnr8/' + repo + '/pulls')
				var requestNum = result.match(/"number": \d+/)[0].split(' ')[1]
				if (!requestNum) throw result
				
				e('git config --local go.' + branch + '.haspullrequest 1')
				echo('Pull request #' + requestNum + ' created successfully.')
			} catch(exception) {
				echo('Error: API request failed.  Message:\n', exception)
				exit(1)
			}
		},
		
		save: function(targetBranch) {
			assertChanges()
			var fs = require('fs')
			
			// Copy the files:
			var modifiedFiles = e('git status | grep "modified:"').replace(/\n/g, '').split(/modified:/g).reduce(function(mem, next) {
				if (!next.trim()) return // Filter out blank entries
				mem ? mem.push(next.trim()) : (mem = [next.trim()])
				return mem
			}, []),
				fileData = {}
			
			echo('Copying files')
			for (var i = 0; i < modifiedFiles.length; i++) {
				fs.readFile(modifiedFiles[i], function(err, data) {
					fileData[modifiedFiles[Object.keys(fileData).length]] = data
					if (Object.keys(fileData).length === modifiedFiles.length) switchBranch()
				})
			}
			function switchBranch(createNew) {
				if (!targetBranch) {
					prompt('Enter the name of the target branch:', function(input) {
						if (!branchExists(input)) {
							prompt('No branch "' + input + '" exists.  Do you want to create it?', function(input) {
								if (!/[Yy]/.test(input)) {
									echo('Exiting...')
									exit(0)
								}
								targetBranch = input
								switchBranch(true)
							})
							return
						}
						targetBranch = input
						switchBranch()
					})
					return
				}
				
				echo('Undoing your changes on this branch')
				e('git checkout -- .')
				if (createNew) {
					echo('Creating new branch "' + targetBranch + '"')
					e('go -n ' + targetBranch)
				} else {
					echo('Switching branches')
					e('git checkout ' + targetBranch)
				}
				pasteFiles()
			}
			function pasteFiles() {
				echo('Inserting your modifications')
				var count = 0;
				for (var i in fileData) {
					fs.writeFile(i, fileData[i], function(err) {
						if (err) {
							echo('Couldn\'t migrate all the files.')
							exit(1)
						}
						if (count === Object.keys(fileData).length) {
							echo('Success!  All files migrated successfully')
							exit(0)
						}
					})
				}
			}
		},
		
		test: function(params) {
			echo(params)
		},
		
		update: function() {
			echo('Syncing master and dev branches...')
			
			try {
				e_throw('git checkout dev')
				e('git pull && git checkout master && git pull')
			} catch(exception) {
				e('git checkout master && git pull && git checkout dev && git pull && git checkout master')
			}
			e('git fetch -p')
			echo('Success!  All up-to-date')
			switchToLocal()
		}
	}
}

new Go([].slice.call(process.argv, 2))




// = = = = = = = = = = = =   Global Functions   = = = = = = = = = = = = //
function assertChanges() {
	if (!hasChanges()) {
		echo('Nothing to commit.  Exiting...')
		exit(0)
	}
}

function assertGitConfig() {
	assertOnLocal()
	if (e('git config --local go.' + currentBranch() + '.commitnum') === '') { // git config doesn't exist.  Create it.
		createGitConfig()
	}
}

function assertOnLocal() {
	if (!onLocal()) {
		if (hasChanges()) {
			echo('You made changes on the ' + currentBranch() + ' branch!  Naughty human...')
			echo('Go cry and run "go -s" to move your changes to a local branch, then try again.')
		} else {
			echo('You\'re not on a local branch.  Let\'s move to one.')
			switchToLocal()
		}
		exit(0)
	}
}

function branchExists(branch) {
	var branches = getLocalBranches()
	return branches.indexOf(branch) > -1
}

function createGitConfig() {
	e('git config --local go.' + currentBranch() + '.commitnum 1 && git config --local go.' + currentBranch() + '.haspullrequest 0')
}

function currentBranch() {
	return e('git rev-parse --abbrev-ref HEAD')
}

// e(command) -- silently executes the given command, returning the trimmed output.
function e(command) {
	return exec(command, {silent: true}).output.trim()
}

function e_throw(command) {
	var result = e(command)
	if (result.indexOf('error:') > -1) throw result
	return result
}

function getCommitMessage() {
	assertGitConfig()
	return currentBranch() + ' #' + e('git config --local go.' + currentBranch() + '.commitnum')
}

function getFirstLocalBranch() {
	return onLocal() ? currentBranch() : getLocalBranches()[0]
}

function getLocalBranches() {
	return e('git rev-parse --abbrev-ref --branches | grep -P "^((?!dev)(?!master).)"').split(/\n/g).filter(function(item) { return item.trim() })
}

function hasChanges() {
	return e('git status | grep "modified:"').length > 0
}

function localExists() {
	return getLocalBranches().length > 0
}

function onLocal() {
	return e('git rev-parse --abbrev-ref HEAD | grep -P "^((?!dev)(?!master).)"') !== ''
}

function prompt(question, callback) {
	var stdin = process.stdin, stdout = process.stdout
	stdin.resume()
	stdout.write(question + ' ')
	
	stdin.once('data', function(data) {
		callback(data.toString().trim())
	})
}

function switchToLocal() {
	if (!localExists()) {
		echo('You don\'t have a local feature branch.  Create one with "go -n [branch]"')
		return
	}
	if (onLocal()) return
	
	// Now, if we're not on a local branch, move to the first one.
	var firstLocalBranch = getFirstLocalBranch()
	e('git checkout ' + firstLocalBranch)
	echo('Moved to local feature branch, "' + firstLocalBranch + '"')
}
