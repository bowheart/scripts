#!/usr/bin/env node
require('shelljs/global')


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
			var params = this.getParams()
			this.options.evaluate(nextArg.slice(1), params)
		}
	},
	
	getParams: function() {
		var params = []
		while (this.args.length && this.args[0].slice(0, 1) !== '-') {
			params.push(this.args.shift())
		}
		return params
	},
	
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
			assertChanges()
			assertOnLocal()
			message || (message = getCommitMessage())
			
			echo('Adding files to commit')
			e('git add --all')
			echo('Committing')
			e('git commit -m "' + message + '"')
			echo('Merging into dev branch')
			e('git checkout dev && git pull && git merge ' + currentBranch())
			echo('Pushing dev branch')
			e('git push')
			echo('Checking out "' + currentBranch() + '" again')
			
			e('git config --local ' + escapeBranch() + '.commitnum $((`git config --local $(escapeBranch).commitnum` + 1))')
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
						e_throw('git branch -d ' + branch) // Delete local branch
						e_throw('git push origin :' + branch) // Delete upstream branch
					} catch(e) {
						echo('Exception thrown: ', e)
						echo('Kurt must have deleted the upstream branch... Cleaning up Kurt\'s generosity...')
						e('git fetch -p')
					}
					e('git config --local --unset ' + branch + '.commitnum && git config --local --unset ' + branch + '.haspullrequest')
					echo('Success!  Branch "' + branch + '" successfully deleted')
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
			echo('Checking out master')
			e('git checkout master')
			echo('Updating master')
			e('git pull')
			echo('Creating new branch "' + branchName + '" and switching to it')
			e('git checkout -b ' + branchName)
			echo('Setting upstream target to "origin/' + branchName + '"')
			e('git push origin ' + branchName + ' && git branch --set-upstream-to origin/' + branchName)
			createGitConfig()
			echo('Success!  You are safely on your new branch')
		},
		
		request: function(requestMessage) {
			requestMessage || (requestMessage = 'Ohdajoiz\'a bein\'Kurtz')
			switchToLocal()
			echo('Creating pull request for branch "' + currentBranch() + '"')
			try {
				var repo = e("git remote show origin | grep 'Fetch URL' | cut -d'/' -f 5 | cut -d'.' -f 1"),
					branch = currentBranch(),
					data = '{"title": "' + branch + '", "head": "' + branch + '", "base": "master", "body": "' + requestMessage + '"}'
				e('curl -so "/var/log/github_pullrequest.log" -X POST -d "' + data + '" -H "Authorization: token `git config --global user.token`" https://api.github.com/repos/ipartnr8/$REPO/pulls')
				e('git config --local ' + branch + '.haspullrequest 1')
				echo('Pull request created successfully')
			} catch(e) {
				echo('Error: API request failed.  Message: ', e)
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
			echo(branchExists('use_core_frms'))
		},
		
		update: function() {
			echo('Syncing master and dev branches...')
			
			var text = '....'
			var interval = setInterval(function() {
				echo(text)
				text += '.'
			}, 200)
			
			try {
				e_throw('git checkout dev')
				e('git pull && git checkout master && git pull')
			} catch(e) {
				e('git checkout master && git pull && git checkout dev && git pull && git checkout master')
			}
			e('git fetch -p')
			clearInterval(interval)
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
	if (e('git config --local ' + escapeBranch() + '.commitnum') === '') { // git config doesn't exist.  Create it.
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
	e('git config --local ' + escapeBranch() + '.commitNum 1 && git config --local ' + escapeBranch() + '.haspullrequest 0')
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
	if (result.indexOf('error:') > -1) throw(result)
	return result
}

function escapeBranch(branch) {
	branch || (branch = currentBranch())
	return branch.replace(/_/g, 'n') // n -- any arbitrary char; git config keys can't have underscores.
}

function getCommitMessage() {
	assertGitConfig()
	return currentBranch() + ' #' + e('git config --local ' + escapeBranch() + '.commitnum')
}

function getFirstLocalBranch() {
	return onLocal() ? currentBranch() : getLocalBranches().split(/\n/g)[0]
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
